package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"code-runner/internal/database/models"
	"code-runner/internal/services"

	"github.com/google/uuid"
)

// DatabaseEventHandler implements EventHandler to persist pipeline events to database
type DatabaseEventHandler struct {
	executionService *services.ExecutionService
	logger           Logger
}

// NewDatabaseEventHandler creates a new database event handler
func NewDatabaseEventHandler(executionService *services.ExecutionService, logger Logger) EventHandler {
	return &DatabaseEventHandler{
		executionService: executionService,
		logger:           logger,
	}
}

// HandleEvent processes pipeline events and persists relevant information to database
func (h *DatabaseEventHandler) HandleEvent(event *PipelineEvent) error {
	if h.executionService == nil {
		return nil // Skip if no execution service is available
	}

	// Get database execution ID from execution data
	var dbExecutionID uuid.UUID
	if event.Data != nil && event.Data.DatabaseExecutionID != uuid.Nil {
		dbExecutionID = event.Data.DatabaseExecutionID
	} else {
		// Try to parse execution ID as UUID (fallback)
		if parsedID, err := uuid.Parse(event.ExecutionID); err == nil {
			dbExecutionID = parsedID
		} else {
			// If we can't find a valid UUID, skip database operations
			h.logWarning("No valid database execution ID found for event", map[string]interface{}{
				"event_type":   event.Type,
				"execution_id": event.ExecutionID,
			})
			return nil
		}
	}

	switch event.Type {
	case PipelineEventStarted:
		return h.handlePipelineStarted(dbExecutionID, event)

	case PipelineEventStepStarted:
		return h.handleStepStarted(dbExecutionID, event)

	case PipelineEventStepCompleted:
		return h.handleStepCompleted(dbExecutionID, event)

	case PipelineEventStepFailed:
		return h.handleStepFailed(dbExecutionID, event)

	case PipelineEventStepSkipped:
		return h.handleStepSkipped(dbExecutionID, event)

	case PipelineEventCompleted:
		return h.handlePipelineCompleted(dbExecutionID, event)

	case PipelineEventFailed:
		return h.handlePipelineFailed(dbExecutionID, event)

	default:
		h.logDebug("Unhandled event type", map[string]interface{}{
			"event_type":   event.Type,
			"execution_id": event.ExecutionID,
		})
	}

	return nil
}

// handlePipelineStarted handles pipeline started events
func (h *DatabaseEventHandler) handlePipelineStarted(executionID uuid.UUID, event *PipelineEvent) error {
	// Add log entry for pipeline start
	err := h.executionService.AddLog(executionID, "info", "Pipeline execution started", "pipeline")
	if err != nil {
		h.logError("Failed to add pipeline start log", err)
		return err
	}

	h.logDebug("Pipeline started event handled", map[string]interface{}{
		"execution_id": executionID,
	})

	return nil
}

// handleStepStarted handles step started events
func (h *DatabaseEventHandler) handleStepStarted(executionID uuid.UUID, event *PipelineEvent) error {
	// Create execution step record
	step, err := h.executionService.AddExecutionStep(executionID, event.StepName, h.getStepOrder(event))
	if err != nil {
		h.logError("Failed to create execution step", err)
		return err
	}

	// Start the step
	err = h.executionService.StartExecutionStep(step.ID)
	if err != nil {
		h.logError("Failed to start execution step", err)
		return err
	}

	// Add log entry
	err = h.executionService.AddLog(executionID, "info",
		fmt.Sprintf("Step '%s' started", event.StepName), "pipeline")
	if err != nil {
		h.logError("Failed to add step start log", err)
	}

	h.logDebug("Step started event handled", map[string]interface{}{
		"execution_id": executionID,
		"step_name":    event.StepName,
		"step_id":      step.ID,
	})

	return nil
}

// handleStepCompleted handles step completed events
func (h *DatabaseEventHandler) handleStepCompleted(executionID uuid.UUID, event *PipelineEvent) error {
	// Find the step to update
	steps, err := h.executionService.GetExecutionSteps(executionID)
	if err != nil {
		h.logError("Failed to get execution steps", err)
		return err
	}

	var stepID uuid.UUID
	for _, step := range steps {
		if step.StepName == event.StepName && step.Status == models.StatusRunning {
			stepID = step.ID
			break
		}
	}

	if stepID == uuid.Nil {
		h.logWarning("Could not find running step to complete", map[string]interface{}{
			"execution_id": executionID,
			"step_name":    event.StepName,
		})
		return nil
	}

	// Prepare metadata
	metadata := h.buildStepMetadata(event)

	// Complete the step
	err = h.executionService.CompleteExecutionStep(stepID, metadata)
	if err != nil {
		h.logError("Failed to complete execution step", err)
		return err
	}

	// Add log entry
	err = h.executionService.AddLog(executionID, "info",
		fmt.Sprintf("Step '%s' completed successfully", event.StepName), "pipeline")
	if err != nil {
		h.logError("Failed to add step completion log", err)
	}

	h.logDebug("Step completed event handled", map[string]interface{}{
		"execution_id": executionID,
		"step_name":    event.StepName,
		"step_id":      stepID,
	})

	return nil
}

// handleStepFailed handles step failed events
func (h *DatabaseEventHandler) handleStepFailed(executionID uuid.UUID, event *PipelineEvent) error {
	// Find the step to update
	steps, err := h.executionService.GetExecutionSteps(executionID)
	if err != nil {
		h.logError("Failed to get execution steps", err)
		return err
	}

	var stepID uuid.UUID
	for _, step := range steps {
		if step.StepName == event.StepName && step.Status == models.StatusRunning {
			stepID = step.ID
			break
		}
	}

	if stepID == uuid.Nil {
		h.logWarning("Could not find running step to fail", map[string]interface{}{
			"execution_id": executionID,
			"step_name":    event.StepName,
		})
		return nil
	}

	// Prepare error message
	errorMessage := event.Message
	if event.Error != nil {
		errorMessage = fmt.Sprintf("%s: %v", errorMessage, event.Error)
	}

	// Fail the step
	err = h.executionService.FailExecutionStep(stepID, errorMessage)
	if err != nil {
		h.logError("Failed to fail execution step", err)
		return err
	}

	// Add error log entry
	err = h.executionService.AddLog(executionID, "error",
		fmt.Sprintf("Step '%s' failed: %s", event.StepName, errorMessage), "pipeline")
	if err != nil {
		h.logError("Failed to add step failure log", err)
	}

	h.logDebug("Step failed event handled", map[string]interface{}{
		"execution_id": executionID,
		"step_name":    event.StepName,
		"step_id":      stepID,
		"error":        errorMessage,
	})

	return nil
}

// handleStepSkipped handles step skipped events
func (h *DatabaseEventHandler) handleStepSkipped(executionID uuid.UUID, event *PipelineEvent) error {
	// Add log entry for skipped step
	err := h.executionService.AddLog(executionID, "info",
		fmt.Sprintf("Step '%s' skipped", event.StepName), "pipeline")
	if err != nil {
		h.logError("Failed to add step skip log", err)
		return err
	}

	h.logDebug("Step skipped event handled", map[string]interface{}{
		"execution_id": executionID,
		"step_name":    event.StepName,
	})

	return nil
}

// handlePipelineCompleted handles pipeline completed events
func (h *DatabaseEventHandler) handlePipelineCompleted(executionID uuid.UUID, event *PipelineEvent) error {
	// Pipeline completion is handled in the main execution flow
	// Just add a completion log here
	err := h.executionService.AddLog(executionID, "info", "Pipeline execution completed successfully", "pipeline")
	if err != nil {
		h.logError("Failed to add pipeline completion log", err)
		return err
	}

	h.logDebug("Pipeline completed event handled", map[string]interface{}{
		"execution_id": executionID,
	})

	return nil
}

// handlePipelineFailed handles pipeline failed events
func (h *DatabaseEventHandler) handlePipelineFailed(executionID uuid.UUID, event *PipelineEvent) error {
	// Pipeline failure is handled in the main execution flow
	// Just add a failure log here
	errorMessage := event.Message
	if event.Error != nil {
		errorMessage = fmt.Sprintf("%s: %v", errorMessage, event.Error)
	}

	err := h.executionService.AddLog(executionID, "error",
		fmt.Sprintf("Pipeline execution failed: %s", errorMessage), "pipeline")
	if err != nil {
		h.logError("Failed to add pipeline failure log", err)
		return err
	}

	h.logDebug("Pipeline failed event handled", map[string]interface{}{
		"execution_id": executionID,
		"error":        errorMessage,
	})

	return nil
}

// buildStepMetadata builds metadata JSON for a step
func (h *DatabaseEventHandler) buildStepMetadata(event *PipelineEvent) string {
	metadata := map[string]interface{}{
		"event_type": event.Type,
		"timestamp":  event.Timestamp,
		"message":    event.Message,
	}

	// Add execution data if available
	if event.Data != nil {
		metadata["language"] = event.Data.Language
		metadata["challenge_id"] = event.Data.ChallengeID
		metadata["student_id"] = event.Data.StudentID

		if event.Data.CompilationResult != nil {
			metadata["compilation_success"] = event.Data.CompilationResult.Success
			metadata["compilation_time_ms"] = event.Data.CompilationResult.CompilationTimeMS
		}

		if len(event.Data.TestResults) > 0 {
			metadata["test_count"] = len(event.Data.TestResults)
		}
	}

	// Convert to JSON
	jsonBytes, err := json.Marshal(metadata)
	if err != nil {
		h.logError("Failed to marshal step metadata", err)
		return "{}"
	}

	return string(jsonBytes)
}

// getStepOrder tries to extract step order from event data
func (h *DatabaseEventHandler) getStepOrder(event *PipelineEvent) int {
	// Try to find step order from execution data
	if event.Data != nil {
		for _, step := range event.Data.CompletedSteps {
			if step.Name == event.StepName {
				return step.Order
			}
		}

		// If not found, use the number of completed steps as order
		return len(event.Data.CompletedSteps)
	}

	// Default order
	return 0
}

// Logging helper methods
func (h *DatabaseEventHandler) logDebug(msg string, fields map[string]interface{}) {
	if h.logger != nil {
		h.logger.Debug(context.Background(), msg, fields)
	}
}

func (h *DatabaseEventHandler) logInfo(msg string, fields map[string]interface{}) {
	if h.logger != nil {
		h.logger.Info(context.Background(), msg, fields)
	}
}

func (h *DatabaseEventHandler) logWarning(msg string, fields map[string]interface{}) {
	if h.logger != nil {
		h.logger.Warn(context.Background(), msg, fields)
	}
}

func (h *DatabaseEventHandler) logError(msg string, err error) {
	if h.logger != nil {
		h.logger.Error(context.Background(), msg, err, nil)
	} else {
		// Fallback to standard logging
		log.Printf("DatabaseEventHandler Error: %s - %v", msg, err)
	}
}
