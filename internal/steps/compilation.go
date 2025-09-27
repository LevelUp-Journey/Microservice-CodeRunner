package steps

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"code-runner/internal/pipeline"
)

// CompilationStep handles compilation of source code for compiled languages
type CompilationStep struct {
	*BaseStep
}

// NewCompilationStep creates a new compilation step
func NewCompilationStep() pipeline.PipelineStep {
	return &CompilationStep{
		BaseStep: NewBaseStep("compilation", 2),
	}
}

// Execute performs compilation of the source code if needed
func (c *CompilationStep) Execute(ctx context.Context, data *pipeline.ExecutionData) error {
	c.AddLog(data, pipeline.LogLevelInfo, "Starting compilation step")

	// Check if language requires compilation
	if !c.requiresCompilation(data.Language) {
		c.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf("Language %s does not require compilation, skipping", data.Language))
		return nil
	}

	// Initialize compilation result
	data.CompilationResult = &pipeline.CompilationResult{
		Success:           false,
		ErrorMessage:      "",
		Warnings:          make([]string, 0),
		CompilationTimeMS: 0,
	}

	// Create working directory if not exists
	if err := c.createWorkingDirectory(data); err != nil {
		return fmt.Errorf("failed to create working directory: %w", err)
	}

	// Write source code to file
	sourceFile, err := c.writeSourceFile(data)
	if err != nil {
		return fmt.Errorf("failed to write source file: %w", err)
	}

	// Add source file to temp files for cleanup
	data.TempFiles = append(data.TempFiles, sourceFile)

	// Perform compilation based on language
	startTime := time.Now()
	outputFile, err := c.compileCode(ctx, data, sourceFile)
	compilationDuration := time.Since(startTime)

	data.CompilationResult.CompilationTimeMS = compilationDuration.Milliseconds()

	if err != nil {
		data.CompilationResult.Success = false
		data.CompilationResult.ErrorMessage = err.Error()
		c.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Compilation failed: %v", err))
		return fmt.Errorf("compilation failed: %w", err)
	}

	// Add compiled output to temp files for cleanup
	if outputFile != "" {
		data.TempFiles = append(data.TempFiles, outputFile)
		c.SetMetadata("compiled_file", outputFile)
	}

	data.CompilationResult.Success = true
	c.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf("Compilation completed successfully in %dms", compilationDuration.Milliseconds()))

	return nil
}

// requiresCompilation checks if the language requires compilation
func (c *CompilationStep) requiresCompilation(language string) bool {
	compiledLanguages := map[string]bool{
		"java":   true,
		"cpp":    true,
		"csharp": true,
		"go":     true,
		"rust":   true,
	}

	return compiledLanguages[language]
}

// createWorkingDirectory creates a temporary working directory
func (c *CompilationStep) createWorkingDirectory(data *pipeline.ExecutionData) error {
	if data.WorkingDirectory != "" {
		return nil // Already created
	}

	tempDir, err := os.MkdirTemp("", fmt.Sprintf("coderunner_%s_", data.ExecutionID))
	if err != nil {
		return err
	}

	data.WorkingDirectory = tempDir
	c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Created working directory: %s", tempDir))

	return nil
}

// writeSourceFile writes the source code to a file
func (c *CompilationStep) writeSourceFile(data *pipeline.ExecutionData) (string, error) {
	fileName := c.getSourceFileName(data.Language, data.ExecutionID)
	filePath := filepath.Join(data.WorkingDirectory, fileName)

	err := os.WriteFile(filePath, []byte(data.Code), 0644)
	if err != nil {
		return "", err
	}

	c.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Source code written to: %s", filePath))
	return filePath, nil
}

// getSourceFileName returns the appropriate source file name for the language
func (c *CompilationStep) getSourceFileName(language, executionID string) string {
	switch language {
	case "java":
		// For Java, we need to extract the class name from the code
		// For now, use a generic name
		return "Solution.java"
	case "cpp":
		return fmt.Sprintf("solution_%s.cpp", executionID)
	case "csharp":
		return fmt.Sprintf("Solution_%s.cs", executionID)
	case "go":
		return fmt.Sprintf("main_%s.go", executionID)
	case "rust":
		return fmt.Sprintf("main_%s.rs", executionID)
	default:
		return fmt.Sprintf("source_%s.txt", executionID)
	}
}

// compileCode compiles the source code based on the language
func (c *CompilationStep) compileCode(ctx context.Context, data *pipeline.ExecutionData, sourceFile string) (string, error) {
	switch data.Language {
	case "java":
		return c.compileJava(ctx, data, sourceFile)
	case "cpp":
		return c.compileCpp(ctx, data, sourceFile)
	case "csharp":
		return c.compileCSharp(ctx, data, sourceFile)
	case "go":
		return c.compileGo(ctx, data, sourceFile)
	case "rust":
		return c.compileRust(ctx, data, sourceFile)
	default:
		return "", fmt.Errorf("compilation not supported for language: %s", data.Language)
	}
}

// compileJava compiles Java source code
func (c *CompilationStep) compileJava(ctx context.Context, data *pipeline.ExecutionData, sourceFile string) (string, error) {
	c.AddLog(data, pipeline.LogLevelInfo, "Compiling Java code")

	// Create compilation context with timeout
	compileCtx, cancel := context.WithTimeout(ctx, time.Duration(data.Config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Run javac
	cmd := exec.CommandContext(compileCtx, "javac", "-d", data.WorkingDirectory, sourceFile)
	cmd.Dir = data.WorkingDirectory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("javac failed: %s", string(output))
	}

	// Check for warnings
	outputStr := string(output)
	if outputStr != "" {
		warnings := c.parseJavaWarnings(outputStr)
		data.CompilationResult.Warnings = warnings
	}

	// Return the class file path (assuming main class is Solution)
	classFile := filepath.Join(data.WorkingDirectory, "Solution.class")
	return classFile, nil
}

// compileCpp compiles C++ source code
func (c *CompilationStep) compileCpp(ctx context.Context, data *pipeline.ExecutionData, sourceFile string) (string, error) {
	c.AddLog(data, pipeline.LogLevelInfo, "Compiling C++ code")

	outputFile := filepath.Join(data.WorkingDirectory, "solution")
	if os.Getenv("OS") == "Windows_NT" {
		outputFile += ".exe"
	}

	// Create compilation context with timeout
	compileCtx, cancel := context.WithTimeout(ctx, time.Duration(data.Config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Run g++
	cmd := exec.CommandContext(compileCtx, "g++", "-o", outputFile, sourceFile, "-std=c++17")
	cmd.Dir = data.WorkingDirectory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("g++ failed: %s", string(output))
	}

	// Check for warnings
	outputStr := string(output)
	if outputStr != "" {
		warnings := c.parseCppWarnings(outputStr)
		data.CompilationResult.Warnings = warnings
	}

	return outputFile, nil
}

// compileCSharp compiles C# source code
func (c *CompilationStep) compileCSharp(ctx context.Context, data *pipeline.ExecutionData, sourceFile string) (string, error) {
	c.AddLog(data, pipeline.LogLevelInfo, "Compiling C# code")

	outputFile := filepath.Join(data.WorkingDirectory, "Solution.exe")

	// Create compilation context with timeout
	compileCtx, cancel := context.WithTimeout(ctx, time.Duration(data.Config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Try mcs first (Mono), then csc (Microsoft)
	var cmd *exec.Cmd
	if c.commandExists("mcs") {
		cmd = exec.CommandContext(compileCtx, "mcs", "-out:"+outputFile, sourceFile)
	} else if c.commandExists("csc") {
		cmd = exec.CommandContext(compileCtx, "csc", "/out:"+outputFile, sourceFile)
	} else {
		return "", fmt.Errorf("no C# compiler found (mcs or csc)")
	}

	cmd.Dir = data.WorkingDirectory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("C# compilation failed: %s", string(output))
	}

	// Check for warnings
	outputStr := string(output)
	if outputStr != "" {
		warnings := c.parseCSharpWarnings(outputStr)
		data.CompilationResult.Warnings = warnings
	}

	return outputFile, nil
}

// compileGo compiles Go source code
func (c *CompilationStep) compileGo(ctx context.Context, data *pipeline.ExecutionData, sourceFile string) (string, error) {
	c.AddLog(data, pipeline.LogLevelInfo, "Compiling Go code")

	outputFile := filepath.Join(data.WorkingDirectory, "solution")
	if os.Getenv("OS") == "Windows_NT" {
		outputFile += ".exe"
	}

	// Create compilation context with timeout
	compileCtx, cancel := context.WithTimeout(ctx, time.Duration(data.Config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Run go build
	cmd := exec.CommandContext(compileCtx, "go", "build", "-o", outputFile, sourceFile)
	cmd.Dir = data.WorkingDirectory

	// Set Go environment variables
	cmd.Env = append(os.Environ(),
		"CGO_ENABLED=0",
		"GOOS="+os.Getenv("GOOS"),
		"GOARCH="+os.Getenv("GOARCH"),
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go build failed: %s", string(output))
	}

	// Go typically doesn't have warnings in the same way, but we can capture any output
	outputStr := string(output)
	if outputStr != "" {
		data.CompilationResult.Warnings = []string{outputStr}
	}

	return outputFile, nil
}

// compileRust compiles Rust source code
func (c *CompilationStep) compileRust(ctx context.Context, data *pipeline.ExecutionData, sourceFile string) (string, error) {
	c.AddLog(data, pipeline.LogLevelInfo, "Compiling Rust code")

	outputFile := filepath.Join(data.WorkingDirectory, "solution")
	if os.Getenv("OS") == "Windows_NT" {
		outputFile += ".exe"
	}

	// Create compilation context with timeout
	compileCtx, cancel := context.WithTimeout(ctx, time.Duration(data.Config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Run rustc
	cmd := exec.CommandContext(compileCtx, "rustc", "-o", outputFile, sourceFile)
	cmd.Dir = data.WorkingDirectory

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("rustc failed: %s", string(output))
	}

	// Check for warnings
	outputStr := string(output)
	if outputStr != "" {
		warnings := c.parseRustWarnings(outputStr)
		data.CompilationResult.Warnings = warnings
	}

	return outputFile, nil
}

// commandExists checks if a command exists in PATH
func (c *CompilationStep) commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// parseJavaWarnings parses Java compiler warnings
func (c *CompilationStep) parseJavaWarnings(output string) []string {
	warnings := make([]string, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings = append(warnings, strings.TrimSpace(line))
		}
	}

	return warnings
}

// parseCppWarnings parses C++ compiler warnings
func (c *CompilationStep) parseCppWarnings(output string) []string {
	warnings := make([]string, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings = append(warnings, strings.TrimSpace(line))
		}
	}

	return warnings
}

// parseCSharpWarnings parses C# compiler warnings
func (c *CompilationStep) parseCSharpWarnings(output string) []string {
	warnings := make([]string, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings = append(warnings, strings.TrimSpace(line))
		}
	}

	return warnings
}

// parseRustWarnings parses Rust compiler warnings
func (c *CompilationStep) parseRustWarnings(output string) []string {
	warnings := make([]string, 0)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if strings.Contains(strings.ToLower(line), "warning") {
			warnings = append(warnings, strings.TrimSpace(line))
		}
	}

	return warnings
}

// CanSkip determines if compilation can be skipped
func (c *CompilationStep) CanSkip(data *pipeline.ExecutionData) bool {
	// Skip if language doesn't require compilation
	return !c.requiresCompilation(data.Language)
}

// Rollback performs cleanup for compilation step
func (c *CompilationStep) Rollback(ctx context.Context, data *pipeline.ExecutionData) error {
	c.AddLog(data, pipeline.LogLevelInfo, "Rolling back compilation step")

	// Remove any compiled files
	if compiledFile, exists := c.GetMetadata("compiled_file"); exists {
		if err := os.Remove(compiledFile); err != nil && !os.IsNotExist(err) {
			c.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Failed to remove compiled file %s: %v", compiledFile, err))
		}
	}

	c.AddLog(data, pipeline.LogLevelInfo, "Compilation step rollback completed")
	return nil
}
