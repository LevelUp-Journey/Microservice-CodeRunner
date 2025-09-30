package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"code-runner/internal/pipeline"
	"code-runner/internal/types"
)

// ExecutionStep handles the actual execution of compiled/interpreted code
type ExecutionStep struct {
	*BaseStep
	maxConcurrentTests int
}

// NewExecutionStep creates a new execution step
func NewExecutionStep() pipeline.PipelineStep {
	return &ExecutionStep{
		BaseStep:           NewBaseStep("execution", 4),
		maxConcurrentTests: 5, // Limit concurrent test executions
	}
}

// Execute runs the code against all test cases
func (e *ExecutionStep) Execute(ctx context.Context, data *pipeline.ExecutionData) error {
	e.AddLog(data, pipeline.LogLevelInfo, "Starting code execution")

	// Get test cases from previous step
	testCases, err := e.getTestCases(data)
	if err != nil {
		return fmt.Errorf("failed to get test cases: %w", err)
	}

	if len(testCases) == 0 {
		return fmt.Errorf("no test cases available for execution")
	}

	// Initialize execution results if not already done
	if len(data.TestResults) == 0 {
		data.TestResults = make([]*pipeline.TestResult, len(testCases))
		for i := range testCases {
			data.TestResults[i] = &pipeline.TestResult{
				TestID:          testCases[i].ID,
				Passed:          false,
				ExpectedOutput:  testCases[i].ExpectedOutput,
				ActualOutput:    "",
				ErrorMessage:    "",
				ExecutionTimeMS: 0,
			}
		}
	}

	// Execute tests
	startTime := time.Now()
	err = e.executeTests(ctx, data, testCases)
	executionDuration := time.Since(startTime)

	// Update execution metrics
	data.ExecutionTimeMS = executionDuration.Milliseconds()
	e.updateExecutionMetrics(data)

	if err != nil {
		e.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Code execution failed: %v", err))
		return fmt.Errorf("code execution failed: %w", err)
	}

	e.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf("Code execution completed in %dms", executionDuration.Milliseconds()))
	return nil
}

// getTestCases retrieves test cases from execution data
func (e *ExecutionStep) getTestCases(data *pipeline.ExecutionData) ([]types.TestCase, error) {
	testCasesJSON, exists := data.Metadata["test_cases_json"]
	if !exists {
		return nil, fmt.Errorf("test cases not found in execution data")
	}

	var testCases []types.TestCase
	if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
		return nil, fmt.Errorf("failed to deserialize test cases: %w", err)
	}

	return testCases, nil
}

// executeTests runs all test cases
func (e *ExecutionStep) executeTests(ctx context.Context, data *pipeline.ExecutionData, testCases []types.TestCase) error {
	// Create execution context with timeout
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(data.Config.TimeoutSeconds)*time.Second)
	defer cancel()

	// Determine execution method based on language
	executor, err := e.createExecutor(data)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	// Execute tests sequentially (can be parallelized later if needed)
	for i, testCase := range testCases {
		select {
		case <-execCtx.Done():
			return fmt.Errorf("execution timeout reached")
		default:
			// Continue with test execution
		}

		e.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Executing test case %s", testCase.ID))

		// Execute single test
		result, err := e.executeSingleTest(execCtx, data, testCase, executor)
		if err != nil {
			e.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Test %s execution failed: %v", testCase.ID, err))
			data.TestResults[i].ErrorMessage = err.Error()
			data.TestResults[i].Passed = false
			continue
		}

		// Update test result
		data.TestResults[i] = result
		e.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Test %s: %s", testCase.ID, map[bool]string{true: "PASSED", false: "FAILED"}[result.Passed]))
	}

	return nil
}

// createExecutor creates an appropriate executor for the language
func (e *ExecutionStep) createExecutor(data *pipeline.ExecutionData) (*CodeExecutor, error) {
	return &CodeExecutor{
		language:         data.Language,
		workingDirectory: data.WorkingDirectory,
		config:           data.Config,
	}, nil
}

// executeSingleTest executes a single test case
func (e *ExecutionStep) executeSingleTest(ctx context.Context, data *pipeline.ExecutionData, testCase types.TestCase, executor *CodeExecutor) (*pipeline.TestResult, error) {
	startTime := time.Now()

	// Create test-specific timeout
	testTimeout := time.Duration(testCase.TimeoutSeconds) * time.Second
	if testTimeout > time.Duration(data.Config.TimeoutSeconds)*time.Second {
		testTimeout = time.Duration(data.Config.TimeoutSeconds) * time.Second
	}

	testCtx, cancel := context.WithTimeout(ctx, testTimeout)
	defer cancel()

	// Execute the code with test input
	output, exitCode, err := executor.Execute(testCtx, testCase.Input, data)
	executionTime := time.Since(startTime)

	result := &pipeline.TestResult{
		TestID:          testCase.ID,
		ExpectedOutput:  testCase.ExpectedOutput,
		ActualOutput:    output,
		ExecutionTimeMS: executionTime.Milliseconds(),
	}

	if err != nil {
		result.ErrorMessage = err.Error()
		result.Passed = false
		return result, nil
	}

	// Compare outputs
	result.Passed = e.compareOutputs(testCase.ExpectedOutput, output)

	// Store exit code in metadata
	if exitCode != 0 {
		result.ErrorMessage = fmt.Sprintf("Process exited with code %d", exitCode)
		result.Passed = false
	}

	return result, nil
}

// compareOutputs compares expected and actual outputs
func (e *ExecutionStep) compareOutputs(expected, actual string) bool {
	// Normalize whitespace
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	// Direct comparison
	if expected == actual {
		return true
	}

	// Normalize line endings
	expected = strings.ReplaceAll(expected, "\r\n", "\n")
	actual = strings.ReplaceAll(actual, "\r\n", "\n")

	return expected == actual
}

// updateExecutionMetrics updates execution metrics in the data
func (e *ExecutionStep) updateExecutionMetrics(data *pipeline.ExecutionData) {
	passedCount := 0
	totalExecutionTime := int64(0)

	for _, result := range data.TestResults {
		if result.Passed {
			passedCount++
		}
		totalExecutionTime += result.ExecutionTimeMS
	}

	// Update approved test IDs
	data.ApprovedTestIDs = make([]string, 0, passedCount)
	for _, result := range data.TestResults {
		if result.Passed {
			data.ApprovedTestIDs = append(data.ApprovedTestIDs, result.TestID)
		}
	}

	// Update success status
	data.Success = passedCount == len(data.TestResults)

	// Update metadata
	data.Metadata["tests_passed"] = strconv.Itoa(passedCount)
	data.Metadata["tests_total"] = strconv.Itoa(len(data.TestResults))
	data.Metadata["tests_failed"] = strconv.Itoa(len(data.TestResults) - passedCount)
	data.Metadata["total_execution_time_ms"] = strconv.FormatInt(totalExecutionTime, 10)

	if data.Success {
		data.Message = fmt.Sprintf("All %d tests passed", len(data.TestResults))
	} else {
		data.Message = fmt.Sprintf("%d of %d tests passed", passedCount, len(data.TestResults))
	}
}

// CanSkip determines if execution can be skipped
func (e *ExecutionStep) CanSkip(data *pipeline.ExecutionData) bool {
	// Never skip execution
	return false
}

// Rollback performs cleanup for execution step
func (e *ExecutionStep) Rollback(ctx context.Context, data *pipeline.ExecutionData) error {
	e.AddLog(data, pipeline.LogLevelInfo, "Rolling back execution step")

	// Clear execution results but keep test structure
	if data.TestResults != nil {
		for i := range data.TestResults {
			data.TestResults[i].Passed = false
			data.TestResults[i].ActualOutput = ""
			data.TestResults[i].ErrorMessage = ""
			data.TestResults[i].ExecutionTimeMS = 0
		}
	}

	// Clear approved test IDs
	data.ApprovedTestIDs = nil
	data.Success = false

	// Clear execution metadata
	delete(data.Metadata, "tests_passed")
	delete(data.Metadata, "tests_total")
	delete(data.Metadata, "tests_failed")
	delete(data.Metadata, "total_execution_time_ms")

	e.AddLog(data, pipeline.LogLevelInfo, "Execution step rollback completed")
	return nil
}

// CodeExecutor handles the actual code execution for different languages
type CodeExecutor struct {
	language         string
	workingDirectory string
	config           *pipeline.ExecutionConfig
}

// Execute runs the code with given input
func (c *CodeExecutor) Execute(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	switch c.language {
	case "javascript":
		return c.executeJavaScript(ctx, input, data)
	case "python":
		return c.executePython(ctx, input, data)
	case "java":
		return c.executeJava(ctx, input, data)
	case "cpp":
		return c.executeCpp(ctx, input, data)
	case "csharp":
		return c.executeCSharp(ctx, input, data)
	case "go":
		return c.executeGo(ctx, input, data)
	case "rust":
		return c.executeRust(ctx, input, data)
	default:
		return "", 1, fmt.Errorf("execution not supported for language: %s", c.language)
	}
}

// executeJavaScript executes JavaScript code
func (c *CodeExecutor) executeJavaScript(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	// Find source file
	sourceFile := filepath.Join(c.workingDirectory, fmt.Sprintf("solution_%s.js", data.ExecutionID))
	if err := os.WriteFile(sourceFile, []byte(data.Code), 0644); err != nil {
		return "", 1, fmt.Errorf("failed to write JavaScript source file: %w", err)
	}

	// Execute with node
	cmd := exec.CommandContext(ctx, "node", sourceFile)
	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// executePython executes Python code
func (c *CodeExecutor) executePython(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	// Find source file
	sourceFile := filepath.Join(c.workingDirectory, fmt.Sprintf("solution_%s.py", data.ExecutionID))
	if err := os.WriteFile(sourceFile, []byte(data.Code), 0644); err != nil {
		return "", 1, fmt.Errorf("failed to write Python source file: %w", err)
	}

	// Execute with python
	var cmd *exec.Cmd
	if c.commandExists("python3") {
		cmd = exec.CommandContext(ctx, "python3", sourceFile)
	} else {
		cmd = exec.CommandContext(ctx, "python", sourceFile)
	}

	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// executeJava executes Java code
func (c *CodeExecutor) executeJava(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	// Execute compiled class
	cmd := exec.CommandContext(ctx, "java", "-cp", c.workingDirectory, "Solution")
	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// executeCpp executes compiled C++ code
func (c *CodeExecutor) executeCpp(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	executable := filepath.Join(c.workingDirectory, "solution")
	if os.Getenv("OS") == "Windows_NT" {
		executable += ".exe"
	}

	cmd := exec.CommandContext(ctx, executable)
	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// executeCSharp executes compiled C# code
func (c *CodeExecutor) executeCSharp(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	executable := filepath.Join(c.workingDirectory, "Solution.exe")

	var cmd *exec.Cmd
	if c.commandExists("mono") {
		cmd = exec.CommandContext(ctx, "mono", executable)
	} else {
		cmd = exec.CommandContext(ctx, executable)
	}

	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// executeGo executes compiled Go code
func (c *CodeExecutor) executeGo(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	executable := filepath.Join(c.workingDirectory, "solution")
	if os.Getenv("OS") == "Windows_NT" {
		executable += ".exe"
	}

	cmd := exec.CommandContext(ctx, executable)
	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// executeRust executes compiled Rust code
func (c *CodeExecutor) executeRust(ctx context.Context, input string, data *pipeline.ExecutionData) (string, int, error) {
	executable := filepath.Join(c.workingDirectory, "solution")
	if os.Getenv("OS") == "Windows_NT" {
		executable += ".exe"
	}

	cmd := exec.CommandContext(ctx, executable)
	cmd.Dir = c.workingDirectory
	cmd.Env = c.buildEnvironment()

	return c.runCommand(cmd, input)
}

// runCommand runs a command with input and returns output and exit code
func (c *CodeExecutor) runCommand(cmd *exec.Cmd, input string) (string, int, error) {
	// Set up stdin with input
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	exitCode := 0

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if status, ok := exitError.Sys().(syscall.WaitStatus); ok {
				exitCode = status.ExitStatus()
			} else {
				exitCode = 1
			}
		} else {
			return "", 1, err
		}
	}

	return string(output), exitCode, nil
}

// buildEnvironment builds the environment variables for execution
func (c *CodeExecutor) buildEnvironment() []string {
	env := os.Environ()

	// Add custom environment variables
	if c.config != nil && c.config.EnvironmentVariables != nil {
		for key, value := range c.config.EnvironmentVariables {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	// Disable network if required
	if c.config != nil && !c.config.EnableNetwork {
		env = append(env, "NO_NETWORK=1")
	}

	return env
}

// commandExists checks if a command exists in PATH
func (c *CodeExecutor) commandExists(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
