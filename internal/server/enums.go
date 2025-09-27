package server

// Enum types (these would normally be generated from protobuf)
type ExecutionStatus int32
type StepStatus int32
type HealthStatus int32
type LogLevel int32

const (
	ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED ExecutionStatus = 0
	ExecutionStatus_EXECUTION_STATUS_PENDING     ExecutionStatus = 1
	ExecutionStatus_EXECUTION_STATUS_RUNNING     ExecutionStatus = 2
	ExecutionStatus_EXECUTION_STATUS_COMPLETED   ExecutionStatus = 3
	ExecutionStatus_EXECUTION_STATUS_FAILED      ExecutionStatus = 4
	ExecutionStatus_EXECUTION_STATUS_TIMEOUT     ExecutionStatus = 5
	ExecutionStatus_EXECUTION_STATUS_CANCELLED   ExecutionStatus = 6

	StepStatus_STEP_STATUS_UNSPECIFIED StepStatus = 0
	StepStatus_STEP_STATUS_PENDING     StepStatus = 1
	StepStatus_STEP_STATUS_RUNNING     StepStatus = 2
	StepStatus_STEP_STATUS_COMPLETED   StepStatus = 3
	StepStatus_STEP_STATUS_FAILED      StepStatus = 4
	StepStatus_STEP_STATUS_SKIPPED     StepStatus = 5

	HealthStatus_HEALTH_STATUS_UNSPECIFIED     HealthStatus = 0
	HealthStatus_HEALTH_STATUS_SERVING         HealthStatus = 1
	HealthStatus_HEALTH_STATUS_NOT_SERVING     HealthStatus = 2
	HealthStatus_HEALTH_STATUS_SERVICE_UNKNOWN HealthStatus = 3

	LogLevel_LOG_LEVEL_UNSPECIFIED LogLevel = 0
	LogLevel_LOG_LEVEL_DEBUG       LogLevel = 1
	LogLevel_LOG_LEVEL_INFO        LogLevel = 2
	LogLevel_LOG_LEVEL_WARN        LogLevel = 3
	LogLevel_LOG_LEVEL_ERROR       LogLevel = 4
)
