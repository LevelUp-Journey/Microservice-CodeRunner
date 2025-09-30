package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"code-runner/internal/codegen"
	"code-runner/internal/database/models"
	"code-runner/internal/database/repository"
	"code-runner/internal/docker"
	"code-runner/internal/pipeline"
	"code-runner/internal/types"
	"code-runner/internal/utils"

	"github.com/google/uuid"
)

// DockerExecutionStep handles code execution using Docker containers
type DockerExecutionStep struct {
	*BaseStep
	dockerExecutor        *docker.DockerExecutor
	generatorFactory      *codegen.GeneratorFactory
	functionParser        *utils.FunctionParser
	generatedTestCodeRepo *repository.GeneratedTestCodeRepository
}

// NewDockerExecutionStep creates a new Docker-based execution step
func NewDockerExecutionStep(logger pipeline.Logger) pipeline.PipelineStep {
	return &DockerExecutionStep{
		BaseStep:         NewBaseStep("docker_execution", 4),
		dockerExecutor:   docker.NewDockerExecutor(logger),
		generatorFactory: codegen.NewGeneratorFactory(),
		functionParser:   utils.NewFunctionParser(),
	}
}

// NewDockerExecutionStepWithRepo creates a new Docker-based execution step with database repository
func NewDockerExecutionStepWithRepo(logger pipeline.Logger, generatedTestCodeRepo *repository.GeneratedTestCodeRepository) pipeline.PipelineStep {
	return &DockerExecutionStep{
		BaseStep:              NewBaseStep("docker_execution", 4),
		dockerExecutor:        docker.NewDockerExecutor(logger),
		generatorFactory:      codegen.NewGeneratorFactory(),
		functionParser:        utils.NewFunctionParser(),
		generatedTestCodeRepo: generatedTestCodeRepo,
	}
}

// Execute runs the code against all test cases using Docker
func (des *DockerExecutionStep) Execute(ctx context.Context, data *pipeline.ExecutionData) error {
	des.AddLog(data, pipeline.LogLevelInfo, "Starting Docker-based code execution")

	// Validate prerequisites
	if err := des.validateExecution(data); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Get test cases from previous step
	testCases, err := des.getTestCases(data)
	if err != nil {
		return fmt.Errorf("failed to get test cases: %w", err)
	}

	// Parse function information from solution code
	functionInfo, err := des.parseFunctionInfo(data)
	if err != nil {
		return fmt.Errorf("failed to parse function information: %w", err)
	}

	// Generate test code
	testCode, err := des.generateTestCode(data, testCases, functionInfo)
	if err != nil {
		return fmt.Errorf("failed to generate test code: %w", err)
	}

	// Execute tests in Docker container
	result, err := des.executeInDocker(ctx, data, testCode)
	if err != nil {
		return fmt.Errorf("Docker execution failed: %w", err)
	}

	// Process results
	des.processExecutionResults(data, result, testCases)

	des.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf("Docker execution completed. Success: %v", result.Success))
	return nil
}

// validateExecution validates execution prerequisites
func (des *DockerExecutionStep) validateExecution(data *pipeline.ExecutionData) error {
	if data.Code == "" {
		return fmt.Errorf("solution code is required")
	}

	if data.Language == "" {
		return fmt.Errorf("programming language is required")
	}

	// Check if language is supported
	supportedLanguages := des.generatorFactory.GetSupportedLanguages()
	isSupported := false
	for _, lang := range supportedLanguages {
		if strings.EqualFold(lang, data.Language) {
			isSupported = true
			break
		}
	}

	if !isSupported {
		return fmt.Errorf("unsupported language: %s. Supported: %v", data.Language, supportedLanguages)
	}

	return nil
}

// getTestCases retrieves test cases from execution data
func (des *DockerExecutionStep) getTestCases(data *pipeline.ExecutionData) ([]types.TestCase, error) {
	testCasesJSON, exists := data.Metadata["test_cases_json"]
	if !exists {
		return nil, fmt.Errorf("test cases not found in execution data")
	}

	var testCases []types.TestCase
	if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
		return nil, fmt.Errorf("failed to deserialize test cases: %w", err)
	}

	// Validate test cases
	validTestCases := make([]types.TestCase, 0, len(testCases))
	for i, testCase := range testCases {
		if !testCase.IsValid() {
			des.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Skipping invalid test case %d: %s", i+1, testCase.ID))
			continue
		}
		validTestCases = append(validTestCases, testCase)
	}

	if len(validTestCases) == 0 {
		return nil, fmt.Errorf("no valid test cases found")
	}

	return validTestCases, nil
}

// parseFunctionInfo extracts function information from solution code
func (des *DockerExecutionStep) parseFunctionInfo(data *pipeline.ExecutionData) (*utils.FunctionInfo, error) {
	des.AddLog(data, pipeline.LogLevelDebug, "Parsing function information from solution code")

	// Try to find main function
	mainFunction, err := des.functionParser.GetMainFunction(data.Code, data.Language)
	if err != nil {
		des.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Could not identify main function: %v", err))

		// Fall back to parsing all functions and using the first one
		functions, parseErr := des.functionParser.ParseFunctions(data.Code, data.Language)
		if parseErr != nil || len(functions) == 0 {
			return nil, fmt.Errorf("no functions found in solution code: %w", parseErr)
		}
		mainFunction = &functions[0]
	}

	des.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf("Identified main function: %s", mainFunction.Name))
	return mainFunction, nil
}

// generateTestCode generates the complete test code including solution and tests
func (des *DockerExecutionStep) generateTestCode(data *pipeline.ExecutionData, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error) {
	startTime := time.Now()
	des.AddLog(data, pipeline.LogLevelDebug, "Generating test code")

	// Get appropriate generator for the language
	generator, err := des.generatorFactory.GetGenerator(data.Language)
	if err != nil {
		return "", fmt.Errorf("failed to get code generator: %w", err)
	}

	// Generate the complete test code
	testCode, err := generator.GenerateTestCode(data.Code, testCases, functionInfo)
	if err != nil {
		return "", fmt.Errorf("failed to generate test code: %w", err)
	}

	generationTime := time.Since(startTime)
	des.AddLog(data, pipeline.LogLevelDebug, fmt.Sprintf("Generated test code (%d bytes) in %v", len(testCode), generationTime))

	// Save generated test code to database if repository is available
	if des.generatedTestCodeRepo != nil && data.DatabaseExecutionID != uuid.Nil {
		if err := des.saveGeneratedTestCode(data, testCode, generator, generationTime, testCases); err != nil {
			des.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Failed to save generated test code to database: %v", err))
			// Don't fail the execution if database save fails
		} else {
			des.AddLog(data, pipeline.LogLevelDebug, "Successfully saved generated test code to database")
		}
	}

	return testCode, nil
}

// executeInDocker executes the test code in a Docker container
func (des *DockerExecutionStep) executeInDocker(ctx context.Context, data *pipeline.ExecutionData, testCode string) (*docker.ExecutionResult, error) {
	des.AddLog(data, pipeline.LogLevelInfo, "Preparing Docker execution")

	// Get generator for execution command
	generator, err := des.generatorFactory.GetGenerator(data.Language)
	if err != nil {
		return nil, fmt.Errorf("failed to get generator: %w", err)
	}

	// Prepare container configuration
	config := &docker.ContainerConfig{
		Language:        data.Language,
		ImageName:       des.dockerExecutor.GetImageName(data.Language),
		MemoryLimitMB:   data.Config.MemoryLimitMB,
		TimeoutSeconds:  data.Config.TimeoutSeconds,
		NetworkDisabled: !data.Config.EnableNetwork,
		ReadOnlyMode:    true,
		EnvironmentVars: data.Config.EnvironmentVariables,
	}

	// Prepare files for execution
	files := map[string]string{
		des.getMainFileName(generator): testCode,
	}

	// Add additional configuration files if needed
	des.addLanguageSpecificFiles(files, data.Language)

	// Execute in Docker
	des.AddLog(data, pipeline.LogLevelInfo, "Executing tests in Docker container")
	result, err := des.dockerExecutor.ExecuteCode(ctx, config, generator.GetExecutionCommand(), files)
	if err != nil {
		return nil, fmt.Errorf("Docker execution failed: %w", err)
	}

	return result, nil
}

// getMainFileName returns the main file name for the language
func (des *DockerExecutionStep) getMainFileName(generator codegen.TestGenerator) string {
	switch strings.ToLower(generator.GetLanguage()) {
	case "cpp":
		return "main.cpp"
	case "python":
		return "test_main.py"
	case "javascript":
		return "test.js"
	case "java":
		return "src/test/java/com/levelup/SolutionTest.java"
	case "go":
		return "main_test.go"
	default:
		return fmt.Sprintf("main%s", generator.GetFileExtension())
	}
}

// addLanguageSpecificFiles adds additional files needed for specific languages
func (des *DockerExecutionStep) addLanguageSpecificFiles(files map[string]string, language string) {
	switch strings.ToLower(language) {
	case "java":
		// Add solution file
		files["src/main/java/com/levelup/Solution.java"] = "" // Will be included in test file
	case "javascript":
		// Jest configuration is already in the Docker image
	}
}

// processExecutionResults processes Docker execution results and updates pipeline data
func (des *DockerExecutionStep) processExecutionResults(data *pipeline.ExecutionData, result *docker.ExecutionResult, testCases []types.TestCase) {
	// Update execution metrics
	data.ExecutionTimeMS = result.ExecutionTimeMS
	data.MemoryUsedMB = result.MemoryUsedMB
	data.ExitCode = result.ExitCode
	data.StandardOutput = result.StandardOutput
	data.StandardError = result.StandardError

	// Determine overall success
	data.Success = result.Success

	// Parse test results from output (this would be enhanced based on testing framework output)
	des.parseTestResults(data, result, testCases)

	// Update status
	if data.Success {
		data.Status = pipeline.ExecutionStatusCompleted
		des.AddLog(data, pipeline.LogLevelInfo, "All tests completed successfully")
	} else {
		data.Status = pipeline.ExecutionStatusFailed
		des.AddLog(data, pipeline.LogLevelError, "Test execution failed")
	}
}

// parseTestResults extracts individual test results from execution output
func (des *DockerExecutionStep) parseTestResults(data *pipeline.ExecutionData, result *docker.ExecutionResult, testCases []types.TestCase) {
	// Initialize test results if not already done
	if len(data.TestResults) == 0 {
		data.TestResults = make([]*pipeline.TestResult, len(testCases))
	}

	// This is a simplified parser - in production, you'd parse the actual test framework output
	// For now, assume all tests passed if the overall execution was successful
	for i, testCase := range testCases {
		if i >= len(data.TestResults) {
			data.TestResults = append(data.TestResults, &pipeline.TestResult{})
		}

		data.TestResults[i] = &pipeline.TestResult{
			TestID:          testCase.ID,
			Passed:          result.Success && result.ExitCode == 0,
			ExpectedOutput:  testCase.ExpectedOutput,
			ActualOutput:    result.StandardOutput,
			ErrorMessage:    result.StandardError,
			ExecutionTimeMS: result.ExecutionTimeMS,
		}

		if !data.TestResults[i].Passed {
			des.AddLog(data, pipeline.LogLevelWarn, fmt.Sprintf("Test %s failed", testCase.ID))
		}
	}

	// Count passed tests
	passedCount := 0
	for _, testResult := range data.TestResults {
		if testResult.Passed {
			passedCount++
		}
	}

	data.Message = fmt.Sprintf("Passed %d/%d tests", passedCount, len(testCases))
}

// saveGeneratedTestCode saves the generated test code to the database
func (des *DockerExecutionStep) saveGeneratedTestCode(data *pipeline.ExecutionData, testCode string, generator codegen.TestGenerator, generationTime time.Duration, testCases []types.TestCase) error {
	// Count custom validation tests
	customValidationCount := 0
	for _, testCase := range testCases {
		if testCase.HasCustomValidation() {
			customValidationCount++
		}
	}

	// Create generated test code record
	generatedTestCode := &models.GeneratedTestCode{
		ExecutionID:         data.DatabaseExecutionID,
		Language:            data.Language,
		GeneratorType:       generator.GetLanguage(),
		TestCode:            testCode,
		ChallengeID:         data.ChallengeID,
		TestCasesCount:      len(testCases),
		HasCustomValidation: customValidationCount > 0,
		GenerationTimeMS:    generationTime.Milliseconds(),
		CodeSizeBytes:       len(testCode),
	}

	// Save to database
	if err := des.generatedTestCodeRepo.Create(generatedTestCode); err != nil {
		return fmt.Errorf("failed to save generated test code: %w", err)
	}

	return nil
}
