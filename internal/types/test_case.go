package types

import "strings"

// ProgrammingLanguage maps to proto enum
type ProgrammingLanguage int32

const (
	ProgrammingLanguageUnspecified ProgrammingLanguage = 0
	ProgrammingLanguageJavascript  ProgrammingLanguage = 1
	ProgrammingLanguagePython      ProgrammingLanguage = 2
	ProgrammingLanguageJava        ProgrammingLanguage = 3
	ProgrammingLanguageCpp         ProgrammingLanguage = 4
	ProgrammingLanguageCsharp      ProgrammingLanguage = 5
	ProgrammingLanguageGo          ProgrammingLanguage = 6
	ProgrammingLanguageRust        ProgrammingLanguage = 7
	ProgrammingLanguageTypescript  ProgrammingLanguage = 8
)

// String returns string representation
func (pl ProgrammingLanguage) String() string {
	switch pl {
	case ProgrammingLanguageJavascript:
		return "javascript"
	case ProgrammingLanguagePython:
		return "python"
	case ProgrammingLanguageJava:
		return "java"
	case ProgrammingLanguageCpp:
		return "c_plus_plus"
	case ProgrammingLanguageCsharp:
		return "csharp"
	case ProgrammingLanguageGo:
		return "go"
	case ProgrammingLanguageRust:
		return "rust"
	case ProgrammingLanguageTypescript:
		return "typescript"
	default:
		return "unspecified"
	}
}

// FromString converts string to ProgrammingLanguage
func ProgrammingLanguageFromString(lang string) ProgrammingLanguage {
	switch strings.ToLower(lang) {
	case "javascript", "js":
		return ProgrammingLanguageJavascript
	case "python", "py":
		return ProgrammingLanguagePython
	case "java":
		return ProgrammingLanguageJava
	case "c_plus_plus", "cpp", "c++":
		return ProgrammingLanguageCpp
	case "csharp", "c#":
		return ProgrammingLanguageCsharp
	case "go", "golang":
		return ProgrammingLanguageGo
	case "rust", "rs":
		return ProgrammingLanguageRust
	case "typescript", "ts":
		return ProgrammingLanguageTypescript
	default:
		return ProgrammingLanguageUnspecified
	}
}

// TestCase represents a test case for execution
// Extended beyond proto to support internal processing with custom validation
type TestCase struct {
	ID                   string `json:"id"`
	Input                string `json:"input"`
	ExpectedOutput       string `json:"expected_output"`
	CustomValidationCode string `json:"custom_validation_code,omitempty"`
	Description          string `json:"description,omitempty"`
	IsPublic             bool   `json:"is_public,omitempty"`
	TimeoutSeconds       int    `json:"timeout_seconds,omitempty"`
}

// HasCustomValidation checks if test case uses custom validation
func (tc *TestCase) HasCustomValidation() bool {
	return strings.TrimSpace(tc.CustomValidationCode) != ""
}

// IsValid validates that test case has either input/output or custom validation
func (tc *TestCase) IsValid() bool {
	hasStandardTest := tc.Input != "" && tc.ExpectedOutput != ""
	hasCustomTest := tc.HasCustomValidation()
	return hasStandardTest || hasCustomTest
}

// ToProtoTestResult converts to proto TestResult format
func (tc *TestCase) ToProtoTestResult(passed bool, actualOutput, errorMessage string, executionTimeMs int64) *TestResult {
	return &TestResult{
		TestID:          tc.ID,
		Passed:          passed,
		ExpectedOutput:  tc.ExpectedOutput,
		ActualOutput:    actualOutput,
		ErrorMessage:    errorMessage,
		ExecutionTimeMS: executionTimeMs,
	}
}

// TestResult matches the proto TestResult exactly
type TestResult struct {
	TestID          string `json:"test_id"`
	Passed          bool   `json:"passed"`
	ExpectedOutput  string `json:"expected_output"`
	ActualOutput    string `json:"actual_output"`
	ErrorMessage    string `json:"error_message"`
	ExecutionTimeMS int64  `json:"execution_time_ms"`
}

// CompilationInfo matches the proto CompilationInfo exactly
type CompilationInfo struct {
	Success           bool     `json:"success"`
	ErrorMessage      string   `json:"error_message"`
	Warnings          []string `json:"warnings"`
	CompilationTimeMS int64    `json:"compilation_time_ms"`
}
