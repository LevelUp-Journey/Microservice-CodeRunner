package steps

import (
	"context"
	"fmt"
	"strings"

	"code-runner/internal/pipeline"
)

// ValidationStep validates the input data before execution
type ValidationStep struct {
	*BaseStep
}

// NewValidationStep creates a new validation step
func NewValidationStep() pipeline.PipelineStep {
	return &ValidationStep{
		BaseStep: NewBaseStep("validation", 1),
	}
}

// Execute performs validation of the execution data
func (v *ValidationStep) Execute(ctx context.Context, data *pipeline.ExecutionData) error {
	v.AddLog(data, pipeline.LogLevelInfo, "Starting input validation")

	// Basic data validation
	if err := v.ValidateData(data); err != nil {
		v.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Basic validation failed: %v", err))
		return fmt.Errorf("validation failed: %w", err)
	}

	// Language validation
	if err := v.validateLanguage(data); err != nil {
		v.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Language validation failed: %v", err))
		return err
	}

	// Code validation
	if err := v.validateCode(data); err != nil {
		v.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Code validation failed: %v", err))
		return err
	}

	// Configuration validation
	if err := v.validateConfiguration(data); err != nil {
		v.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Configuration validation failed: %v", err))
		return err
	}

	// Normalize language
	data.Language = NormalizeLanguage(strings.ToLower(data.Language))

	v.AddLog(data, pipeline.LogLevelInfo, "Input validation completed successfully")
	return nil
}

// validateLanguage validates the programming language
func (v *ValidationStep) validateLanguage(data *pipeline.ExecutionData) error {
	if data.Language == "" {
		return fmt.Errorf("programming language is required")
	}

	// Normalize and check if language is supported
	normalizedLang := NormalizeLanguage(strings.ToLower(data.Language))
	if !IsLanguageSupported(normalizedLang) {
		return fmt.Errorf("unsupported programming language: %s. Supported languages: %s",
			data.Language, strings.Join(GetSupportedLanguages(), ", "))
	}

	v.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Language validated: %s", normalizedLang))
	return nil
}

// validateCode validates the source code
func (v *ValidationStep) validateCode(data *pipeline.ExecutionData) error {
	if strings.TrimSpace(data.Code) == "" {
		return fmt.Errorf("source code cannot be empty")
	}

	// Check code length (max 10MB)
	const maxCodeSize = 10 * 1024 * 1024
	if len(data.Code) > maxCodeSize {
		return fmt.Errorf("source code too large: %d bytes (max %d bytes)", len(data.Code), maxCodeSize)
	}

	// Basic syntax checks per language
	if err := v.performBasicSyntaxCheck(data); err != nil {
		return err
	}

	v.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Code validated: %d characters", len(data.Code)))
	return nil
}

// validateConfiguration validates the execution configuration
func (v *ValidationStep) validateConfiguration(data *pipeline.ExecutionData) error {
	if data.Config == nil {
		// Set default configuration
		data.Config = &pipeline.ExecutionConfig{
			TimeoutSeconds:       30,
			MemoryLimitMB:        512,
			EnableNetwork:        false,
			EnvironmentVariables: make(map[string]string),
			DebugMode:            false,
		}
		v.AddLog(data, pipeline.LogLevelInfo, "Using default execution configuration")
		return nil
	}

	// Validate timeout
	if data.Config.TimeoutSeconds <= 0 {
		data.Config.TimeoutSeconds = 30
	} else if data.Config.TimeoutSeconds > 300 { // Max 5 minutes
		return fmt.Errorf("timeout too large: %d seconds (max 300 seconds)", data.Config.TimeoutSeconds)
	}

	// Validate memory limit
	if data.Config.MemoryLimitMB <= 0 {
		data.Config.MemoryLimitMB = 512
	} else if data.Config.MemoryLimitMB > 2048 { // Max 2GB
		return fmt.Errorf("memory limit too large: %d MB (max 2048 MB)", data.Config.MemoryLimitMB)
	}

	// Validate environment variables
	if data.Config.EnvironmentVariables == nil {
		data.Config.EnvironmentVariables = make(map[string]string)
	}

	// Limit number of environment variables
	if len(data.Config.EnvironmentVariables) > 50 {
		return fmt.Errorf("too many environment variables: %d (max 50)", len(data.Config.EnvironmentVariables))
	}

	// Validate environment variable names and values
	for key, value := range data.Config.EnvironmentVariables {
		if key == "" {
			return fmt.Errorf("environment variable name cannot be empty")
		}
		if len(key) > 100 {
			return fmt.Errorf("environment variable name too long: %s (max 100 characters)", key)
		}
		if len(value) > 1000 {
			return fmt.Errorf("environment variable value too long for key %s (max 1000 characters)", key)
		}
		// Check for dangerous environment variables
		if v.isDangerousEnvVar(key) {
			return fmt.Errorf("dangerous environment variable not allowed: %s", key)
		}
	}

	v.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Configuration validated: timeout=%ds, memory=%dMB",
		data.Config.TimeoutSeconds, data.Config.MemoryLimitMB))
	return nil
}

// performBasicSyntaxCheck performs basic syntax validation per language
func (v *ValidationStep) performBasicSyntaxCheck(data *pipeline.ExecutionData) error {
	code := strings.TrimSpace(data.Code)
	language := NormalizeLanguage(strings.ToLower(data.Language))

	switch language {
	case "javascript", "typescript":
		return v.validateJavaScriptSyntax(code)
	case "python":
		return v.validatePythonSyntax(code)
	case "java":
		return v.validateJavaSyntax(code)
	case "cpp":
		return v.validateCppSyntax(code)
	case "csharp":
		return v.validateCSharpSyntax(code)
	case "go":
		return v.validateGoSyntax(code)
	case "rust":
		return v.validateRustSyntax(code)
	default:
		// For unsupported languages, just check if code is not empty
		return nil
	}
}

// validateJavaScriptSyntax performs basic JavaScript syntax validation
func (v *ValidationStep) validateJavaScriptSyntax(code string) error {
	// Basic checks for JavaScript
	if !strings.Contains(code, "function") && !strings.Contains(code, "=>") && !strings.Contains(code, "const") && !strings.Contains(code, "let") && !strings.Contains(code, "var") {
		v.AddLog(nil, pipeline.LogLevelWarn, "JavaScript code may not contain any function or variable declarations")
	}

	// Check for basic syntax errors
	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("mismatched curly braces in JavaScript code")
	}

	if strings.Count(code, "(") != strings.Count(code, ")") {
		return fmt.Errorf("mismatched parentheses in JavaScript code")
	}

	return nil
}

// validatePythonSyntax performs basic Python syntax validation
func (v *ValidationStep) validatePythonSyntax(code string) error {
	lines := strings.Split(code, "\n")

	// Check for basic Python structure
	hasFunction := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "def ") {
			hasFunction = true
			break
		}
	}

	if !hasFunction {
		v.AddLog(nil, pipeline.LogLevelWarn, "Python code may not contain any function definitions")
	}

	return nil
}

// validateJavaSyntax performs basic Java syntax validation
func (v *ValidationStep) validateJavaSyntax(code string) error {
	if !strings.Contains(code, "class") && !strings.Contains(code, "interface") {
		return fmt.Errorf("Java code must contain at least one class or interface")
	}

	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("mismatched curly braces in Java code")
	}

	return nil
}

// validateCppSyntax performs basic C++ syntax validation
func (v *ValidationStep) validateCppSyntax(code string) error {
	if !strings.Contains(code, "#include") {
		v.AddLog(nil, pipeline.LogLevelWarn, "C++ code may not contain any include statements")
	}

	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("mismatched curly braces in C++ code")
	}

	return nil
}

// validateCSharpSyntax performs basic C# syntax validation
func (v *ValidationStep) validateCSharpSyntax(code string) error {
	if !strings.Contains(code, "using") && !strings.Contains(code, "namespace") && !strings.Contains(code, "class") {
		v.AddLog(nil, pipeline.LogLevelWarn, "C# code may not follow standard structure")
	}

	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("mismatched curly braces in C# code")
	}

	return nil
}

// validateGoSyntax performs basic Go syntax validation
func (v *ValidationStep) validateGoSyntax(code string) error {
	if !strings.Contains(code, "package") {
		return fmt.Errorf("Go code must contain a package declaration")
	}

	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("mismatched curly braces in Go code")
	}

	return nil
}

// validateRustSyntax performs basic Rust syntax validation
func (v *ValidationStep) validateRustSyntax(code string) error {
	if !strings.Contains(code, "fn") {
		v.AddLog(nil, pipeline.LogLevelWarn, "Rust code may not contain any function definitions")
	}

	if strings.Count(code, "{") != strings.Count(code, "}") {
		return fmt.Errorf("mismatched curly braces in Rust code")
	}

	return nil
}

// isDangerousEnvVar checks if an environment variable is potentially dangerous
func (v *ValidationStep) isDangerousEnvVar(key string) bool {
	dangerousVars := []string{
		"PATH", "LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH",
		"HOME", "USER", "SHELL", "TERM",
		"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY",
		"GOOGLE_APPLICATION_CREDENTIALS",
		"GITHUB_TOKEN", "GITLAB_TOKEN",
		"DATABASE_URL", "DB_PASSWORD",
		"PRIVATE_KEY", "SECRET_KEY", "API_KEY",
	}

	keyUpper := strings.ToUpper(key)
	for _, dangerous := range dangerousVars {
		if keyUpper == dangerous || strings.Contains(keyUpper, "PASSWORD") || strings.Contains(keyUpper, "SECRET") {
			return true
		}
	}

	return false
}

// CanSkip determines if validation can be skipped (never skip validation)
func (v *ValidationStep) CanSkip(data *pipeline.ExecutionData) bool {
	return false
}

// Rollback performs cleanup for validation step (nothing to rollback)
func (v *ValidationStep) Rollback(ctx context.Context, data *pipeline.ExecutionData) error {
	v.AddLog(data, pipeline.LogLevelInfo, "Validation step rollback completed (no action required)")
	return nil
}
