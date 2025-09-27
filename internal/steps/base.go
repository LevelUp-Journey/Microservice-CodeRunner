package steps

import (
	"context"
	"fmt"
	"time"

	"code-runner/internal/pipeline"
)

// BaseStep provides common functionality for all pipeline steps
type BaseStep struct {
	name     string
	order    int
	metadata map[string]string
}

// NewBaseStep creates a new base step
func NewBaseStep(name string, order int) *BaseStep {
	return &BaseStep{
		name:     name,
		order:    order,
		metadata: make(map[string]string),
	}
}

// GetName returns the name of the step
func (b *BaseStep) GetName() string {
	return b.name
}

// GetOrder returns the execution order of this step
func (b *BaseStep) GetOrder() int {
	return b.order
}

// CanSkip determines if this step can be skipped based on current state
// Default implementation never skips - override in specific steps
func (b *BaseStep) CanSkip(data *pipeline.ExecutionData) bool {
	return false
}

// Rollback performs cleanup if the step needs to be rolled back
// Default implementation does nothing - override in specific steps
func (b *BaseStep) Rollback(ctx context.Context, data *pipeline.ExecutionData) error {
	return nil
}

// SetMetadata sets metadata for the step
func (b *BaseStep) SetMetadata(key, value string) {
	if b.metadata == nil {
		b.metadata = make(map[string]string)
	}
	b.metadata[key] = value
}

// GetMetadata gets metadata for the step
func (b *BaseStep) GetMetadata(key string) (string, bool) {
	if b.metadata == nil {
		return "", false
	}
	value, exists := b.metadata[key]
	return value, exists
}

// AddLog adds a log entry to the execution data
func (b *BaseStep) AddLog(data *pipeline.ExecutionData, level pipeline.LogLevel, message string) {
	if data.Logs == nil {
		data.Logs = make([]pipeline.LogEntry, 0)
	}

	logEntry := pipeline.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		StepName:  b.name,
		Metadata:  make(map[string]string),
	}

	// Copy step metadata to log entry
	for k, v := range b.metadata {
		logEntry.Metadata[k] = v
	}

	data.Logs = append(data.Logs, logEntry)
}

// ValidateData validates that required execution data is present
func (b *BaseStep) ValidateData(data *pipeline.ExecutionData) error {
	if data == nil {
		return fmt.Errorf("execution data cannot be nil")
	}

	if data.ExecutionID == "" {
		return fmt.Errorf("execution ID is required")
	}

	if data.SolutionID == "" {
		return fmt.Errorf("solution ID is required")
	}

	if data.Code == "" {
		return fmt.Errorf("code is required")
	}

	if data.Language == "" {
		return fmt.Errorf("language is required")
	}

	return nil
}

// GetSupportedLanguages returns the list of supported programming languages
func GetSupportedLanguages() []string {
	return []string{
		"javascript",
		"python",
		"java",
		"cpp",
		"c++",
		"csharp",
		"c#",
		"go",
		"rust",
		"typescript",
	}
}

// IsLanguageSupported checks if a language is supported
func IsLanguageSupported(language string) bool {
	supported := GetSupportedLanguages()
	for _, lang := range supported {
		if lang == language {
			return true
		}
	}
	return false
}

// NormalizeLanguage normalizes language names to standard format
func NormalizeLanguage(language string) string {
	switch language {
	case "js", "javascript", "node", "nodejs":
		return "javascript"
	case "py", "python", "python3":
		return "python"
	case "java":
		return "java"
	case "cpp", "c++", "cxx":
		return "cpp"
	case "csharp", "c#", "cs", "dotnet":
		return "csharp"
	case "go", "golang":
		return "go"
	case "rust", "rs":
		return "rust"
	case "ts", "typescript":
		return "typescript"
	default:
		return language
	}
}

// StepResult represents the result of a step execution
type StepResult struct {
	Success   bool
	Message   string
	Error     error
	Metadata  map[string]string
	Duration  time.Duration
	StartTime time.Time
	EndTime   time.Time
}

// NewStepResult creates a new step result
func NewStepResult() *StepResult {
	return &StepResult{
		Metadata:  make(map[string]string),
		StartTime: time.Now(),
	}
}

// SetSuccess marks the step as successful
func (sr *StepResult) SetSuccess(message string) {
	sr.Success = true
	sr.Message = message
	sr.EndTime = time.Now()
	sr.Duration = sr.EndTime.Sub(sr.StartTime)
}

// SetFailure marks the step as failed
func (sr *StepResult) SetFailure(message string, err error) {
	sr.Success = false
	sr.Message = message
	sr.Error = err
	sr.EndTime = time.Now()
	sr.Duration = sr.EndTime.Sub(sr.StartTime)
}

// AddMetadata adds metadata to the step result
func (sr *StepResult) AddMetadata(key, value string) {
	if sr.Metadata == nil {
		sr.Metadata = make(map[string]string)
	}
	sr.Metadata[key] = value
}
