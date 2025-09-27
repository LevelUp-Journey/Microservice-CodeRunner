package steps

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"code-runner/internal/pipeline"
)

// TestFetchingStep fetches test cases for the solution (simplified with mock data)
type TestFetchingStep struct {
	*BaseStep
}

// TestCase represents a test case for execution
type TestCase struct {
	ID             string `json:"id"`
	Input          string `json:"input"`
	ExpectedOutput string `json:"expected_output"`
	Description    string `json:"description,omitempty"`
	IsPublic       bool   `json:"is_public,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`
}

// NewTestFetchingStep creates a new test fetching step
func NewTestFetchingStep(challengesAPIURL string) pipeline.PipelineStep {
	return &TestFetchingStep{
		BaseStep: NewBaseStep("test_fetching", 3),
	}
}

// Execute fetches test cases for the solution
func (t *TestFetchingStep) Execute(ctx context.Context, data *pipeline.ExecutionData) error {
	t.AddLog(data, pipeline.LogLevelInfo, "Starting test case fetching")

	// Validate required data
	if data.ChallengeID == "" {
		return fmt.Errorf("challenge ID is required for test fetching")
	}

	// Generate mock test cases based on language and challenge ID
	testCases := t.generateMockTestCases(data.Language, data.ChallengeID)

	// Store test cases in execution data
	if err := t.storeTestCases(data, testCases); err != nil {
		t.AddLog(data, pipeline.LogLevelError, fmt.Sprintf("Failed to store test cases: %v", err))
		return fmt.Errorf("failed to store test cases: %w", err)
	}

	t.AddLog(data, pipeline.LogLevelInfo, fmt.Sprintf("Successfully generated %d test cases", len(testCases)))
	return nil
}

// generateMockTestCases creates mock test cases based on language and challenge
func (t *TestFetchingStep) generateMockTestCases(language, challengeID string) []TestCase {
	// Generate different test cases based on challenge type
	switch challengeID {
	case "factorial", "challenge_456":
		return []TestCase{
			{
				ID:             "test_1",
				Input:          "5",
				ExpectedOutput: "120",
				Description:    "Calculate factorial of 5",
				IsPublic:       true,
				TimeoutSeconds: 30,
			},
			{
				ID:             "test_2",
				Input:          "0",
				ExpectedOutput: "1",
				Description:    "Calculate factorial of 0",
				IsPublic:       true,
				TimeoutSeconds: 30,
			},
			{
				ID:             "test_3",
				Input:          "10",
				ExpectedOutput: "3628800",
				Description:    "Calculate factorial of 10",
				IsPublic:       false,
				TimeoutSeconds: 30,
			},
		}
	case "sum", "addition":
		return []TestCase{
			{
				ID:             "sum_test_1",
				Input:          "2 3",
				ExpectedOutput: "5",
				Description:    "Add two numbers: 2 + 3",
				IsPublic:       true,
				TimeoutSeconds: 15,
			},
			{
				ID:             "sum_test_2",
				Input:          "10 -5",
				ExpectedOutput: "5",
				Description:    "Add positive and negative: 10 + (-5)",
				IsPublic:       true,
				TimeoutSeconds: 15,
			},
			{
				ID:             "sum_test_3",
				Input:          "0 0",
				ExpectedOutput: "0",
				Description:    "Add zeros: 0 + 0",
				IsPublic:       false,
				TimeoutSeconds: 15,
			},
		}
	case "reverse", "string_reverse":
		return []TestCase{
			{
				ID:             "reverse_test_1",
				Input:          "hello",
				ExpectedOutput: "olleh",
				Description:    "Reverse string: hello",
				IsPublic:       true,
				TimeoutSeconds: 20,
			},
			{
				ID:             "reverse_test_2",
				Input:          "JavaScript",
				ExpectedOutput: "tpircSavaJ",
				Description:    "Reverse string: JavaScript",
				IsPublic:       true,
				TimeoutSeconds: 20,
			},
			{
				ID:             "reverse_test_3",
				Input:          "12345",
				ExpectedOutput: "54321",
				Description:    "Reverse numeric string",
				IsPublic:       false,
				TimeoutSeconds: 20,
			},
		}
	case "hello_world", "print":
		return []TestCase{
			{
				ID:             "hello_test_1",
				Input:          "",
				ExpectedOutput: "Hello World",
				Description:    "Print Hello World",
				IsPublic:       true,
				TimeoutSeconds: 10,
			},
		}
	default:
		// Default simple test cases
		return []TestCase{
			{
				ID:             "default_test_1",
				Input:          "test",
				ExpectedOutput: "test",
				Description:    "Echo input",
				IsPublic:       true,
				TimeoutSeconds: 30,
			},
			{
				ID:             "default_test_2",
				Input:          "hello",
				ExpectedOutput: "hello",
				Description:    "Echo hello",
				IsPublic:       false,
				TimeoutSeconds: 30,
			},
		}
	}
}

// storeTestCases stores test cases in execution data
func (t *TestFetchingStep) storeTestCases(data *pipeline.ExecutionData, testCases []TestCase) error {
	if len(testCases) == 0 {
		return fmt.Errorf("no test cases generated")
	}

	// Initialize test results
	data.TestResults = make([]*pipeline.TestResult, len(testCases))

	for i, testCase := range testCases {
		data.TestResults[i] = &pipeline.TestResult{
			TestID:          testCase.ID,
			Passed:          false,
			ExpectedOutput:  testCase.ExpectedOutput,
			ActualOutput:    "",
			ErrorMessage:    "",
			ExecutionTimeMS: 0,
		}
	}

	// Store test case metadata
	data.Metadata["test_cases_count"] = strconv.Itoa(len(testCases))

	// Count public vs private tests
	publicCount := 0
	for _, testCase := range testCases {
		if testCase.IsPublic {
			publicCount++
		}
	}
	data.Metadata["public_tests_count"] = strconv.Itoa(publicCount)
	data.Metadata["private_tests_count"] = strconv.Itoa(len(testCases) - publicCount)

	// Store raw test cases in metadata for execution step
	testCasesJSON, err := json.Marshal(testCases)
	if err != nil {
		return fmt.Errorf("failed to serialize test cases: %w", err)
	}
	data.Metadata["test_cases_json"] = string(testCasesJSON)

	return nil
}

// GetTestCasesFromData extracts test cases from execution data
func (t *TestFetchingStep) GetTestCasesFromData(data *pipeline.ExecutionData) ([]TestCase, error) {
	testCasesJSON, exists := data.Metadata["test_cases_json"]
	if !exists {
		return nil, fmt.Errorf("test cases not found in execution data")
	}

	var testCases []TestCase
	if err := json.Unmarshal([]byte(testCasesJSON), &testCases); err != nil {
		return nil, fmt.Errorf("failed to deserialize test cases: %w", err)
	}

	return testCases, nil
}

// CanSkip determines if test fetching can be skipped
func (t *TestFetchingStep) CanSkip(data *pipeline.ExecutionData) bool {
	// Skip if we already have test results (re-run scenario)
	if len(data.TestResults) > 0 {
		return true
	}

	// Skip if challenge ID is empty (validation should catch this)
	if data.ChallengeID == "" {
		return false
	}

	return false
}

// Rollback performs cleanup for test fetching step
func (t *TestFetchingStep) Rollback(ctx context.Context, data *pipeline.ExecutionData) error {
	t.AddLog(data, pipeline.LogLevelInfo, "Rolling back test fetching step")

	// Clear test-related data
	data.TestResults = nil

	// Clear test-related metadata
	delete(data.Metadata, "test_cases_json")
	delete(data.Metadata, "test_cases_count")
	delete(data.Metadata, "public_tests_count")
	delete(data.Metadata, "private_tests_count")

	t.AddLog(data, pipeline.LogLevelInfo, "Test fetching step rollback completed")
	return nil
}
