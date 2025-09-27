package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BaseModel provides common fields for all models
type BaseModel struct {
	ID        uuid.UUID      `gorm:"type:uuid;primary_key;" json:"id"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}

// BeforeCreate hook to generate UUID before creating record
func (base *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if base.ID == uuid.Nil {
		base.ID = uuid.New()
	}
	return nil
}

// ExecutionStatus represents the status of a code execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusTimedOut  ExecutionStatus = "timed_out"
	StatusCancelled ExecutionStatus = "cancelled"
)

// Execution represents a code execution record
type Execution struct {
	BaseModel

	// Request Information
	SolutionID  string `gorm:"type:varchar(255);not null;index" json:"solution_id"`
	ChallengeID string `gorm:"type:varchar(255);not null;index" json:"challenge_id"`
	StudentID   string `gorm:"type:varchar(255);not null;index" json:"student_id"`
	Language    string `gorm:"type:varchar(50);not null" json:"language"`

	// Execution Details
	Status          ExecutionStatus `gorm:"type:varchar(50);not null;default:'pending';index" json:"status"`
	Code            string          `gorm:"type:text;not null" json:"code"`
	ExecutionTimeMs *int64          `json:"execution_time_ms,omitempty"`
	MemoryUsageMB   *float64        `json:"memory_usage_mb,omitempty"`

	// Results
	Success         bool     `gorm:"default:false" json:"success"`
	Message         string   `gorm:"type:text" json:"message,omitempty"`
	ApprovedTestIDs []string `gorm:"type:json" json:"approved_test_ids,omitempty"`
	FailedTestIDs   []string `gorm:"type:json" json:"failed_test_ids,omitempty"`
	TotalTests      int      `gorm:"default:0" json:"total_tests"`
	PassedTests     int      `gorm:"default:0" json:"passed_tests"`

	// Error Information
	ErrorMessage     string `gorm:"type:text" json:"error_message,omitempty"`
	ErrorType        string `gorm:"type:varchar(100)" json:"error_type,omitempty"`
	CompilationError string `gorm:"type:text" json:"compilation_error,omitempty"`
	RuntimeError     string `gorm:"type:text" json:"runtime_error,omitempty"`

	// Metadata
	ServerInstance string          `gorm:"type:varchar(255)" json:"server_instance,omitempty"`
	ClientIP       string          `gorm:"type:varchar(45)" json:"client_ip,omitempty"`
	UserAgent      string          `gorm:"type:varchar(500)" json:"user_agent,omitempty"`
	ExecutionSteps []ExecutionStep `gorm:"foreignKey:ExecutionID" json:"execution_steps,omitempty"`
	ExecutionLogs  []ExecutionLog  `gorm:"foreignKey:ExecutionID" json:"execution_logs,omitempty"`
	TestResults    []TestResult    `gorm:"foreignKey:ExecutionID" json:"test_results,omitempty"`
}

// ExecutionStep represents each step in the execution pipeline
type ExecutionStep struct {
	BaseModel

	ExecutionID  uuid.UUID       `gorm:"type:uuid;not null;index" json:"execution_id"`
	StepName     string          `gorm:"type:varchar(100);not null" json:"step_name"`
	StepOrder    int             `gorm:"not null" json:"step_order"`
	Status       ExecutionStatus `gorm:"type:varchar(50);not null" json:"status"`
	StartedAt    *time.Time      `json:"started_at,omitempty"`
	CompletedAt  *time.Time      `json:"completed_at,omitempty"`
	DurationMs   *int64          `json:"duration_ms,omitempty"`
	ErrorMessage string          `gorm:"type:text" json:"error_message,omitempty"`
	Metadata     string          `gorm:"type:json" json:"metadata,omitempty"`

	// Foreign key relation
	Execution Execution `gorm:"foreignKey:ExecutionID" json:"-"`
}

// ExecutionLog represents logs generated during execution
type ExecutionLog struct {
	BaseModel

	ExecutionID uuid.UUID `gorm:"type:uuid;not null;index" json:"execution_id"`
	Level       string    `gorm:"type:varchar(20);not null;index" json:"level"` // info, warn, error, debug
	Message     string    `gorm:"type:text;not null" json:"message"`
	Source      string    `gorm:"type:varchar(100)" json:"source,omitempty"` // compiler, runtime, system
	Timestamp   time.Time `gorm:"not null;index" json:"timestamp"`

	// Foreign key relation
	Execution Execution `gorm:"foreignKey:ExecutionID" json:"-"`
}

// TestResult represents the result of individual test cases
type TestResult struct {
	BaseModel

	ExecutionID     uuid.UUID `gorm:"type:uuid;not null;index" json:"execution_id"`
	TestID          string    `gorm:"type:varchar(255);not null" json:"test_id"`
	TestName        string    `gorm:"type:varchar(255)" json:"test_name,omitempty"`
	Input           string    `gorm:"type:text" json:"input,omitempty"`
	ExpectedOutput  string    `gorm:"type:text" json:"expected_output,omitempty"`
	ActualOutput    string    `gorm:"type:text" json:"actual_output,omitempty"`
	Passed          bool      `gorm:"default:false" json:"passed"`
	ExecutionTimeMs *int64    `json:"execution_time_ms,omitempty"`
	ErrorMessage    string    `gorm:"type:text" json:"error_message,omitempty"`

	// Foreign key relation
	Execution Execution `gorm:"foreignKey:ExecutionID" json:"-"`
}

// TableName specifies the table name for Execution
func (Execution) TableName() string {
	return "executions"
}

// TableName specifies the table name for ExecutionStep
func (ExecutionStep) TableName() string {
	return "execution_steps"
}

// TableName specifies the table name for ExecutionLog
func (ExecutionLog) TableName() string {
	return "execution_logs"
}

// TableName specifies the table name for TestResult
func (TestResult) TableName() string {
	return "test_results"
}
