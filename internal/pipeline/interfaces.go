package pipeline

import (
	"context"
	"time"
)

// PipelineStep represents a single step in the execution pipeline
type PipelineStep interface {
	// GetName returns the name of the step
	GetName() string

	// Execute runs the step with the given context and data
	Execute(ctx context.Context, data *ExecutionData) error

	// GetOrder returns the execution order of this step
	GetOrder() int

	// CanSkip determines if this step can be skipped based on current state
	CanSkip(data *ExecutionData) bool

	// Rollback performs cleanup if the step needs to be rolled back
	Rollback(ctx context.Context, data *ExecutionData) error
}

// Pipeline orchestrates the execution of multiple steps
type Pipeline interface {
	// AddStep adds a step to the pipeline
	AddStep(step PipelineStep) error

	// Execute runs all steps in the pipeline
	Execute(ctx context.Context, data *ExecutionData) error

	// GetSteps returns all steps in the pipeline
	GetSteps() []PipelineStep

	// GetExecutionID returns the current execution ID
	GetExecutionID() string
}

// ExecutionData holds all data passed through the pipeline
type ExecutionData struct {
	// Request information
	SolutionID  string
	ChallengeID string
	StudentID   string
	Code        string
	Language    string

	// Configuration
	Config *ExecutionConfig

	// Execution state
	ExecutionID string
	Status      ExecutionStatus
	StartTime   time.Time
	EndTime     time.Time

	// Results
	ApprovedTestIDs []string
	Success         bool
	Message         string
	ErrorMessage    string
	ExitCode        int

	// Performance metrics
	ExecutionTimeMS int64
	MemoryUsedMB    int64

	// Compilation info
	CompilationResult *CompilationResult

	// Test results
	TestResults []*TestResult

	// Step tracking
	CompletedSteps []StepInfo
	CurrentStep    string

	// Additional metadata
	Metadata map[string]string

	// Temporary files and cleanup
	WorkingDirectory string
	TempFiles        []string

	// Output and logs
	StandardOutput string
	StandardError  string
	Logs           []LogEntry
}

// ExecutionConfig holds configuration for code execution
type ExecutionConfig struct {
	TimeoutSeconds       int32
	MemoryLimitMB        int32
	EnableNetwork        bool
	EnvironmentVariables map[string]string
	DebugMode            bool
}

// ExecutionStatus represents the current status of execution
type ExecutionStatus int32

const (
	ExecutionStatusUnspecified ExecutionStatus = iota
	ExecutionStatusPending
	ExecutionStatusRunning
	ExecutionStatusCompleted
	ExecutionStatusFailed
	ExecutionStatusTimeout
	ExecutionStatusCancelled
)

// String returns string representation of ExecutionStatus
func (s ExecutionStatus) String() string {
	switch s {
	case ExecutionStatusPending:
		return "PENDING"
	case ExecutionStatusRunning:
		return "RUNNING"
	case ExecutionStatusCompleted:
		return "COMPLETED"
	case ExecutionStatusFailed:
		return "FAILED"
	case ExecutionStatusTimeout:
		return "TIMEOUT"
	case ExecutionStatusCancelled:
		return "CANCELLED"
	default:
		return "UNSPECIFIED"
	}
}

// CompilationResult holds compilation information
type CompilationResult struct {
	Success           bool
	ErrorMessage      string
	Warnings          []string
	CompilationTimeMS int64
}

// TestResult represents the result of a single test
type TestResult struct {
	TestID          string
	Passed          bool
	ExpectedOutput  string
	ActualOutput    string
	ErrorMessage    string
	ExecutionTimeMS int64
}

// StepInfo holds information about completed pipeline steps
type StepInfo struct {
	Name        string
	Status      StepStatus
	StartedAt   time.Time
	CompletedAt time.Time
	Message     string
	Error       string
	Metadata    map[string]string
	Order       int
}

// StepStatus represents the status of a pipeline step
type StepStatus int32

const (
	StepStatusUnspecified StepStatus = iota
	StepStatusPending
	StepStatusRunning
	StepStatusCompleted
	StepStatusFailed
	StepStatusSkipped
)

// String returns string representation of StepStatus
func (s StepStatus) String() string {
	switch s {
	case StepStatusPending:
		return "PENDING"
	case StepStatusRunning:
		return "RUNNING"
	case StepStatusCompleted:
		return "COMPLETED"
	case StepStatusFailed:
		return "FAILED"
	case StepStatusSkipped:
		return "SKIPPED"
	default:
		return "UNSPECIFIED"
	}
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp time.Time
	Level     LogLevel
	Message   string
	StepName  string
	Metadata  map[string]string
}

// LogLevel represents the level of a log entry
type LogLevel int32

const (
	LogLevelUnspecified LogLevel = iota
	LogLevelDebug
	LogLevelInfo
	LogLevelWarn
	LogLevelError
)

// String returns string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	default:
		return "UNSPECIFIED"
	}
}

// PipelineEventType represents different types of pipeline events
type PipelineEventType int32

const (
	PipelineEventUnspecified PipelineEventType = iota
	PipelineEventStarted
	PipelineEventStepStarted
	PipelineEventStepCompleted
	PipelineEventStepFailed
	PipelineEventStepSkipped
	PipelineEventCompleted
	PipelineEventFailed
)

// PipelineEvent represents an event in the pipeline execution
type PipelineEvent struct {
	Type        PipelineEventType
	Timestamp   time.Time
	ExecutionID string
	StepName    string
	Message     string
	Error       error
	Data        *ExecutionData
}

// EventHandler handles pipeline events
type EventHandler interface {
	HandleEvent(event *PipelineEvent) error
}

// Logger provides logging capabilities for the pipeline
type Logger interface {
	Debug(ctx context.Context, msg string, fields map[string]interface{})
	Info(ctx context.Context, msg string, fields map[string]interface{})
	Warn(ctx context.Context, msg string, fields map[string]interface{})
	Error(ctx context.Context, msg string, err error, fields map[string]interface{})
}

// ExecutionContext provides context for pipeline execution
type ExecutionContext struct {
	Context       context.Context
	Logger        Logger
	EventHandler  EventHandler
	ExecutionData *ExecutionData
}
