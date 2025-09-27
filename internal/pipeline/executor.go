package pipeline

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
)

// DefaultPipeline implements the Pipeline interface
type DefaultPipeline struct {
	executionID  string
	steps        []PipelineStep
	logger       Logger
	eventHandler EventHandler
	mutex        sync.RWMutex
	maxRetries   int
	retryDelay   time.Duration
}

// PipelineConfig holds configuration for pipeline creation
type PipelineConfig struct {
	ExecutionID  string
	Logger       Logger
	EventHandler EventHandler
	MaxRetries   int
	RetryDelay   time.Duration
}

// NewPipeline creates a new pipeline instance
func NewPipeline(config *PipelineConfig) Pipeline {
	if config == nil {
		config = &PipelineConfig{}
	}

	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}

	if config.RetryDelay == 0 {
		config.RetryDelay = time.Second * 2
	}

	return &DefaultPipeline{
		executionID:  config.ExecutionID,
		steps:        make([]PipelineStep, 0),
		logger:       config.Logger,
		eventHandler: config.EventHandler,
		maxRetries:   config.MaxRetries,
		retryDelay:   config.RetryDelay,
	}
}

// AddStep adds a step to the pipeline
func (p *DefaultPipeline) AddStep(step PipelineStep) error {
	if step == nil {
		return fmt.Errorf("step cannot be nil")
	}

	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.steps = append(p.steps, step)

	// Sort steps by order
	sort.Slice(p.steps, func(i, j int) bool {
		return p.steps[i].GetOrder() < p.steps[j].GetOrder()
	})

	return nil
}

// GetSteps returns all steps in the pipeline
func (p *DefaultPipeline) GetSteps() []PipelineStep {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	steps := make([]PipelineStep, len(p.steps))
	copy(steps, p.steps)
	return steps
}

// GetExecutionID returns the current execution ID
func (p *DefaultPipeline) GetExecutionID() string {
	return p.executionID
}

// Execute runs all steps in the pipeline
func (p *DefaultPipeline) Execute(ctx context.Context, data *ExecutionData) error {
	if data == nil {
		return fmt.Errorf("execution data cannot be nil")
	}

	// Initialize execution data
	p.initializeExecutionData(data)

	// Fire pipeline started event
	p.fireEvent(&PipelineEvent{
		Type:        PipelineEventStarted,
		Timestamp:   time.Now(),
		ExecutionID: p.executionID,
		Message:     "Pipeline execution started",
		Data:        data,
	})

	p.logInfo(ctx, "Starting pipeline execution", map[string]interface{}{
		"execution_id": p.executionID,
		"steps_count":  len(p.steps),
		"solution_id":  data.SolutionID,
		"language":     data.Language,
	})

	var lastError error

	// Execute each step
	for _, step := range p.steps {
		select {
		case <-ctx.Done():
			data.Status = ExecutionStatusCancelled
			data.Message = "Execution cancelled"
			p.logInfo(ctx, "Pipeline execution cancelled", nil)
			return ctx.Err()
		default:
			// Continue with step execution
		}

		// Check if step can be skipped
		if step.CanSkip(data) {
			p.handleSkippedStep(ctx, step, data)
			continue
		}

		// Execute step with retry logic
		err := p.executeStepWithRetry(ctx, step, data)
		if err != nil {
			lastError = err
			p.handleFailedStep(ctx, step, data, err)

			// Mark execution as failed and break
			data.Status = ExecutionStatusFailed
			data.Message = fmt.Sprintf("Step '%s' failed: %v", step.GetName(), err)
			break
		}

		p.handleCompletedStep(ctx, step, data)
	}

	// Finalize execution
	p.finalizeExecution(ctx, data, lastError)

	if lastError != nil {
		p.fireEvent(&PipelineEvent{
			Type:        PipelineEventFailed,
			Timestamp:   time.Now(),
			ExecutionID: p.executionID,
			Message:     "Pipeline execution failed",
			Error:       lastError,
			Data:        data,
		})
		return lastError
	}

	p.fireEvent(&PipelineEvent{
		Type:        PipelineEventCompleted,
		Timestamp:   time.Now(),
		ExecutionID: p.executionID,
		Message:     "Pipeline execution completed successfully",
		Data:        data,
	})

	return nil
}

// initializeExecutionData sets up initial execution data
func (p *DefaultPipeline) initializeExecutionData(data *ExecutionData) {
	if data.ExecutionID == "" {
		data.ExecutionID = p.executionID
	}

	data.Status = ExecutionStatusRunning
	data.StartTime = time.Now()
	data.CompletedSteps = make([]StepInfo, 0)

	if data.Metadata == nil {
		data.Metadata = make(map[string]string)
	}

	if data.Logs == nil {
		data.Logs = make([]LogEntry, 0)
	}

	if data.TempFiles == nil {
		data.TempFiles = make([]string, 0)
	}
}

// executeStepWithRetry executes a step with retry logic
func (p *DefaultPipeline) executeStepWithRetry(ctx context.Context, step PipelineStep, data *ExecutionData) error {
	var lastError error

	for attempt := 1; attempt <= p.maxRetries; attempt++ {
		// Update current step
		data.CurrentStep = step.GetName()

		p.logDebug(ctx, "Executing step", map[string]interface{}{
			"step":    step.GetName(),
			"attempt": attempt,
			"order":   step.GetOrder(),
		})

		// Fire step started event
		p.fireEvent(&PipelineEvent{
			Type:        PipelineEventStepStarted,
			Timestamp:   time.Now(),
			ExecutionID: p.executionID,
			StepName:    step.GetName(),
			Message:     fmt.Sprintf("Step '%s' started (attempt %d)", step.GetName(), attempt),
			Data:        data,
		})

		// Execute the step
		startTime := time.Now()
		err := step.Execute(ctx, data)
		duration := time.Since(startTime)

		if err == nil {
			p.logInfo(ctx, "Step completed successfully", map[string]interface{}{
				"step":        step.GetName(),
				"duration_ms": duration.Milliseconds(),
				"attempt":     attempt,
			})
			return nil
		}

		lastError = err
		p.logError(ctx, "Step execution failed", err, map[string]interface{}{
			"step":        step.GetName(),
			"attempt":     attempt,
			"duration_ms": duration.Milliseconds(),
		})

		// If this isn't the last attempt, wait before retrying
		if attempt < p.maxRetries {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(p.retryDelay):
				continue
			}
		}
	}

	return fmt.Errorf("step '%s' failed after %d attempts: %w", step.GetName(), p.maxRetries, lastError)
}

// handleSkippedStep handles a skipped step
func (p *DefaultPipeline) handleSkippedStep(ctx context.Context, step PipelineStep, data *ExecutionData) {
	stepInfo := StepInfo{
		Name:        step.GetName(),
		Status:      StepStatusSkipped,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
		Message:     "Step skipped based on execution conditions",
		Order:       step.GetOrder(),
		Metadata:    make(map[string]string),
	}

	data.CompletedSteps = append(data.CompletedSteps, stepInfo)

	p.logInfo(ctx, "Step skipped", map[string]interface{}{
		"step":  step.GetName(),
		"order": step.GetOrder(),
	})

	p.fireEvent(&PipelineEvent{
		Type:        PipelineEventStepSkipped,
		Timestamp:   time.Now(),
		ExecutionID: p.executionID,
		StepName:    step.GetName(),
		Message:     fmt.Sprintf("Step '%s' skipped", step.GetName()),
		Data:        data,
	})
}

// handleCompletedStep handles a successfully completed step
func (p *DefaultPipeline) handleCompletedStep(ctx context.Context, step PipelineStep, data *ExecutionData) {
	now := time.Now()
	stepInfo := StepInfo{
		Name:        step.GetName(),
		Status:      StepStatusCompleted,
		StartedAt:   now, // We don't track individual step start times in this implementation
		CompletedAt: now,
		Message:     "Step completed successfully",
		Order:       step.GetOrder(),
		Metadata:    make(map[string]string),
	}

	data.CompletedSteps = append(data.CompletedSteps, stepInfo)

	p.fireEvent(&PipelineEvent{
		Type:        PipelineEventStepCompleted,
		Timestamp:   time.Now(),
		ExecutionID: p.executionID,
		StepName:    step.GetName(),
		Message:     fmt.Sprintf("Step '%s' completed successfully", step.GetName()),
		Data:        data,
	})
}

// handleFailedStep handles a failed step
func (p *DefaultPipeline) handleFailedStep(ctx context.Context, step PipelineStep, data *ExecutionData, err error) {
	now := time.Now()
	stepInfo := StepInfo{
		Name:        step.GetName(),
		Status:      StepStatusFailed,
		StartedAt:   now,
		CompletedAt: now,
		Message:     "Step failed",
		Error:       err.Error(),
		Order:       step.GetOrder(),
		Metadata:    make(map[string]string),
	}

	data.CompletedSteps = append(data.CompletedSteps, stepInfo)
	data.ErrorMessage = err.Error()

	p.logError(ctx, "Step failed", err, map[string]interface{}{
		"step":  step.GetName(),
		"order": step.GetOrder(),
	})

	p.fireEvent(&PipelineEvent{
		Type:        PipelineEventStepFailed,
		Timestamp:   time.Now(),
		ExecutionID: p.executionID,
		StepName:    step.GetName(),
		Message:     fmt.Sprintf("Step '%s' failed", step.GetName()),
		Error:       err,
		Data:        data,
	})

	// Attempt to rollback the failed step
	p.attemptStepRollback(ctx, step, data)
}

// attemptStepRollback attempts to rollback a failed step
func (p *DefaultPipeline) attemptStepRollback(ctx context.Context, step PipelineStep, data *ExecutionData) {
	p.logInfo(ctx, "Attempting step rollback", map[string]interface{}{
		"step": step.GetName(),
	})

	if err := step.Rollback(ctx, data); err != nil {
		p.logError(ctx, "Step rollback failed", err, map[string]interface{}{
			"step": step.GetName(),
		})
		// Continue with execution even if rollback fails
	} else {
		p.logInfo(ctx, "Step rollback completed", map[string]interface{}{
			"step": step.GetName(),
		})
	}
}

// finalizeExecution finalizes the pipeline execution
func (p *DefaultPipeline) finalizeExecution(ctx context.Context, data *ExecutionData, lastError error) {
	data.EndTime = time.Now()
	data.ExecutionTimeMS = data.EndTime.Sub(data.StartTime).Milliseconds()

	if lastError == nil && data.Status != ExecutionStatusFailed {
		data.Status = ExecutionStatusCompleted
		data.Success = true
		if data.Message == "" {
			data.Message = "Pipeline execution completed successfully"
		}
	} else {
		data.Success = false
		if data.Status != ExecutionStatusCancelled && data.Status != ExecutionStatusTimeout {
			data.Status = ExecutionStatusFailed
		}
	}

	// Clear current step
	data.CurrentStep = ""

	p.logInfo(ctx, "Pipeline execution finalized", map[string]interface{}{
		"execution_id":    p.executionID,
		"status":          data.Status.String(),
		"success":         data.Success,
		"duration_ms":     data.ExecutionTimeMS,
		"completed_steps": len(data.CompletedSteps),
	})
}

// fireEvent fires a pipeline event if an event handler is configured
func (p *DefaultPipeline) fireEvent(event *PipelineEvent) {
	if p.eventHandler != nil {
		if err := p.eventHandler.HandleEvent(event); err != nil {
			// Log error but don't fail the pipeline
			if p.logger != nil {
				p.logger.Error(context.Background(), "Failed to handle pipeline event", err, map[string]interface{}{
					"event_type":   event.Type,
					"execution_id": event.ExecutionID,
					"step_name":    event.StepName,
				})
			}
		}
	}
}

// Logging helper methods
func (p *DefaultPipeline) logDebug(ctx context.Context, msg string, fields map[string]interface{}) {
	if p.logger != nil {
		if fields == nil {
			fields = make(map[string]interface{})
		}
		fields["execution_id"] = p.executionID
		p.logger.Debug(ctx, msg, fields)
	}
}

func (p *DefaultPipeline) logInfo(ctx context.Context, msg string, fields map[string]interface{}) {
	if p.logger != nil {
		if fields == nil {
			fields = make(map[string]interface{})
		}
		fields["execution_id"] = p.executionID
		p.logger.Info(ctx, msg, fields)
	}
}

func (p *DefaultPipeline) logError(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	if p.logger != nil {
		if fields == nil {
			fields = make(map[string]interface{})
		}
		fields["execution_id"] = p.executionID
		p.logger.Error(ctx, msg, err, fields)
	}
}

// DefaultLogger implements the Logger interface using zap
type DefaultLogger struct {
	logger *zap.Logger
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() (Logger, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, err
	}

	return &DefaultLogger{logger: logger}, nil
}

// Debug logs a debug message
func (l *DefaultLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	zapFields := l.convertFields(fields)
	l.logger.Debug(msg, zapFields...)
}

// Info logs an info message
func (l *DefaultLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	zapFields := l.convertFields(fields)
	l.logger.Info(msg, zapFields...)
}

// Warn logs a warning message
func (l *DefaultLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	zapFields := l.convertFields(fields)
	l.logger.Warn(msg, zapFields...)
}

// Error logs an error message
func (l *DefaultLogger) Error(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	zapFields := l.convertFields(fields)
	if err != nil {
		zapFields = append(zapFields, zap.Error(err))
	}
	l.logger.Error(msg, zapFields...)
}

// convertFields converts map[string]interface{} to zap.Field slice
func (l *DefaultLogger) convertFields(fields map[string]interface{}) []zap.Field {
	if fields == nil {
		return nil
	}

	zapFields := make([]zap.Field, 0, len(fields))
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
	}

	return zapFields
}

// Close closes the logger
func (l *DefaultLogger) Close() error {
	return l.logger.Sync()
}
