package models

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
	ExecutionTimeMS int64           `gorm:"type:bigint" json:"execution_time_ms"`
	MemoryUsageMB   float64         `gorm:"type:decimal(10,2)" json:"memory_usage_mb"`

	// Results
	Success         bool     `gorm:"type:boolean;default:false" json:"success"`
	Message         string   `gorm:"type:text" json:"message"`
	ApprovedTestIDs []string `gorm:"type:jsonb;default:'[]'" json:"approved_test_ids"`
	FailedTestIDs   []string `gorm:"type:jsonb;default:'[]'" json:"failed_test_ids"`
	TotalTests      int      `gorm:"type:integer;default:0" json:"total_tests"`
	PassedTests     int      `gorm:"type:integer;default:0" json:"passed_tests"`

	// Error Information
	ErrorMessage     string `gorm:"type:text" json:"error_message"`
	ErrorType        string `gorm:"type:varchar(100)" json:"error_type"`
	CompilationError string `gorm:"type:text" json:"compilation_error"`
	RuntimeError     string `gorm:"type:text" json:"runtime_error"`

	// Metadata
	ServerInstance string `gorm:"type:varchar(255)" json:"server_instance"`
	ClientIP       string `gorm:"type:varchar(45)" json:"client_ip"`
	UserAgent      string `gorm:"type:varchar(500)" json:"user_agent"`
}

// TableName returns the table name for GORM
func (Execution) TableName() string {
	return "executions"
}
