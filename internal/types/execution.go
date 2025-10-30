package types

import (
	"time"

	"github.com/google/uuid"
)

// ExecutionRequest representa la solicitud de ejecución de código
type ExecutionRequest struct {
	SolutionID    uuid.UUID        `json:"solution_id"`
	ChallengeID   uuid.UUID        `json:"challenge_id"`
	CodeVersionID uuid.UUID        `json:"code_version_id"`
	StudentID     uuid.UUID        `json:"student_id"`
	Code          string           `json:"code"`
	Language      string           `json:"language"`
	Config        *ExecutionConfig `json:"config,omitempty"`
	TestCases     []*TestCase      `json:"test_cases"`
}

// TestCase representa un caso de prueba
type TestCase struct {
	TestID               uuid.UUID `json:"test_id"`
	CodeVersionTestID    uuid.UUID `json:"code_version_test_id"`
	Input                string    `json:"input"`
	ExpectedOutput       string    `json:"expected_output"`
	CustomValidationCode string    `json:"custom_validation_code,omitempty"`
}

// HasCustomValidation verifica si el test case tiene validación personalizada
func (tc *TestCase) HasCustomValidation() bool {
	return tc.CustomValidationCode != ""
}

// ExecutionConfig representa la configuración de ejecución
type ExecutionConfig struct {
	TimeoutSeconds       int64             `json:"timeout_seconds,omitempty"`
	MemoryLimitMB        int64             `json:"memory_limit_mb,omitempty"`
	EnableNetwork        bool              `json:"enable_network,omitempty"`
	EnvironmentVariables map[string]string `json:"environment_variables,omitempty"`
	DebugMode            bool              `json:"debug_mode,omitempty"`
}

// ProgrammingLanguage representa los lenguajes de programación soportados
type ProgrammingLanguage string

const (
	LanguageCpp        ProgrammingLanguage = "c_plus_plus"
	LanguagePython     ProgrammingLanguage = "python"
	LanguageJavaScript ProgrammingLanguage = "javascript"
	LanguageJava       ProgrammingLanguage = "java"
	LanguageGo         ProgrammingLanguage = "go"
)

// String devuelve la representación en string del lenguaje
func (pl ProgrammingLanguage) String() string {
	return string(pl)
}

// IsValid valida si el lenguaje es soportado
func (pl ProgrammingLanguage) IsValid() bool {
	switch pl {
	case LanguageCpp, LanguagePython, LanguageJavaScript, LanguageJava, LanguageGo:
		return true
	default:
		return false
	}
}

// IsValid valida que el test case tenga los campos requeridos
func (tc *TestCase) IsValid() bool {
	return tc.TestID != uuid.Nil && (tc.Input != "" || tc.CustomValidationCode != "")
}

// ExecutionResponse representa la respuesta de ejecución con metadatos detallados
type ExecutionResponse struct {
	ApprovedTestIDs []uuid.UUID        `json:"approved_test_ids"`
	Success         bool               `json:"success"`
	Message         string             `json:"message"`
	ExecutionID     uuid.UUID          `json:"execution_id"`
	Metadata        *ExecutionMetadata `json:"metadata,omitempty"`
	PipelineSteps   []*PipelineStep    `json:"pipeline_steps,omitempty"`
}

// ExecutionMetadata contiene información detallada de la ejecución
type ExecutionMetadata struct {
	StartedAt       *time.Time         `json:"started_at,omitempty"`
	CompletedAt     *time.Time         `json:"completed_at,omitempty"`
	ExecutionTimeMS int64              `json:"execution_time_ms"`
	MemoryUsedMB    int32              `json:"memory_used_mb"`
	ExitCode        int32              `json:"exit_code"`
	Compilation     *CompilationResult `json:"compilation,omitempty"`
	TestResults     []*TestResult      `json:"test_results,omitempty"`
}

// CompilationResult resultado de la compilación para lenguajes compilados
type CompilationResult struct {
	Success           bool     `json:"success"`
	ErrorMessage      string   `json:"error_message,omitempty"`
	Warnings          []string `json:"warnings,omitempty"`
	CompilationTimeMS int64    `json:"compilation_time_ms"`
}

// TestResult resultado individual de un test
type TestResult struct {
	TestID          uuid.UUID `json:"test_id"`
	Passed          bool      `json:"passed"`
	ExpectedOutput  string    `json:"expected_output,omitempty"`
	ActualOutput    string    `json:"actual_output,omitempty"`
	ErrorMessage    string    `json:"error_message,omitempty"`
	ExecutionTimeMS int64     `json:"execution_time_ms"`
}

// PipelineStep información de cada paso del pipeline
type PipelineStep struct {
	Name         string            `json:"name"`
	Status       StepStatus        `json:"status"`
	StartedAt    *time.Time        `json:"started_at,omitempty"`
	CompletedAt  *time.Time        `json:"completed_at,omitempty"`
	Message      string            `json:"message,omitempty"`
	Error        string            `json:"error,omitempty"`
	StepMetadata map[string]string `json:"step_metadata,omitempty"`
	StepOrder    int32             `json:"step_order"`
}

// StepStatus enumeración de estados de los pasos del pipeline
type StepStatus int32

const (
	StepStatusUnknown   StepStatus = 0
	StepStatusPending   StepStatus = 1
	StepStatusRunning   StepStatus = 2
	StepStatusCompleted StepStatus = 3
	StepStatusFailed    StepStatus = 4
	StepStatusSkipped   StepStatus = 5
)
