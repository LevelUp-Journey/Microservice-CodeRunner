package server

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Message types (these would normally be generated from protobuf)

// ExecutionRequest represents a request to execute code
type ExecutionRequest struct {
	SolutionId  string
	ChallengeId string
	StudentId   string
	Code        string
	Language    string
	Config      *ExecutionConfig
}

// ExecutionConfig holds configuration for code execution
type ExecutionConfig struct {
	TimeoutSeconds       int32
	MemoryLimitMb        int32
	EnableNetwork        bool
	EnvironmentVariables map[string]string
	DebugMode            bool
}

// ExecutionResponse represents the response from code execution
type ExecutionResponse struct {
	ApprovedTestIds []string
	Success         bool
	Message         string
	ExecutionId     string
	Metadata        *ExecutionMetadata
	PipelineSteps   []*PipelineStep
}

// ExecutionStatusRequest represents a request for execution status
type ExecutionStatusRequest struct {
	ExecutionId string
}

// ExecutionStatusResponse represents the response with execution status
type ExecutionStatusResponse struct {
	ExecutionId     string
	Status          ExecutionStatus
	ApprovedTestIds []string
	Success         bool
	Message         string
	Metadata        *ExecutionMetadata
	PipelineSteps   []*PipelineStep
}

// HealthCheckRequest represents a health check request
type HealthCheckRequest struct{}

// HealthCheckResponse represents a health check response
type HealthCheckResponse struct {
	Status    HealthStatus
	Message   string
	Timestamp *timestamppb.Timestamp
}

// StreamLogsRequest represents a request to stream logs
type StreamLogsRequest struct {
	ExecutionId string
}

// LogEntry represents a single log entry
type LogEntry struct {
	Timestamp *timestamppb.Timestamp
	Level     LogLevel
	Message   string
	StepName  string
	Metadata  map[string]string
}

// ExecutionMetadata contains metadata about the execution
type ExecutionMetadata struct {
	StartedAt       *timestamppb.Timestamp
	CompletedAt     *timestamppb.Timestamp
	ExecutionTimeMs int64
	MemoryUsedMb    int64
	ExitCode        int32
	Compilation     *CompilationInfo
	TestResults     []*TestResult
}

// CompilationInfo contains information about compilation
type CompilationInfo struct {
	Success           bool
	ErrorMessage      string
	Warnings          []string
	CompilationTimeMs int64
}

// TestResult represents the result of running a test case
type TestResult struct {
	TestId          string
	Passed          bool
	ExpectedOutput  string
	ActualOutput    string
	ErrorMessage    string
	ExecutionTimeMs int64
}

// PipelineStep represents a step in the execution pipeline
type PipelineStep struct {
	Name         string
	Status       StepStatus
	StartedAt    *timestamppb.Timestamp
	CompletedAt  *timestamppb.Timestamp
	Message      string
	Error        string
	StepMetadata map[string]string
	StepOrder    int32
}

// Stream interface
type CodeExecutionService_StreamExecutionLogsServer interface {
	Send(*LogEntry) error
	Context() context.Context
}

// Service interface
type CodeExecutionServiceServer interface {
	ExecuteCode(context.Context, *ExecutionRequest) (*ExecutionResponse, error)
	GetExecutionStatus(context.Context, *ExecutionStatusRequest) (*ExecutionStatusResponse, error)
	HealthCheck(context.Context, *HealthCheckRequest) (*HealthCheckResponse, error)
	StreamExecutionLogs(*StreamLogsRequest, CodeExecutionService_StreamExecutionLogsServer) error
}
