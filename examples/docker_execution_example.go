package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"code-runner/internal/pipeline"
	"code-runner/internal/steps"
	"code-runner/internal/types"
)

// ExampleLogger implements pipeline.Logger for demonstration
type ExampleLogger struct{}

func (l *ExampleLogger) Debug(ctx context.Context, msg string, fields map[string]interface{}) {
	fmt.Printf("[DEBUG] %s %v\n", msg, fields)
}

func (l *ExampleLogger) Info(ctx context.Context, msg string, fields map[string]interface{}) {
	fmt.Printf("[INFO] %s %v\n", msg, fields)
}

func (l *ExampleLogger) Warn(ctx context.Context, msg string, fields map[string]interface{}) {
	fmt.Printf("[WARN] %s %v\n", msg, fields)
}

func (l *ExampleLogger) Error(ctx context.Context, msg string, err error, fields map[string]interface{}) {
	fmt.Printf("[ERROR] %s: %v %v\n", msg, err, fields)
}

func main() {
	// Example usage of the Docker-based code execution system

	logger := &ExampleLogger{}
	ctx := context.Background()

	// Example 1: Python factorial function with standard tests
	fmt.Println("=== Example 1: Python Factorial ===")
	err := runPythonFactorialExample(ctx, logger)
	if err != nil {
		log.Printf("Python example failed: %v", err)
	}

	// Example 2: C++ with custom validation
	fmt.Println("\n=== Example 2: C++ with Custom Validation ===")
	err = runCppCustomValidationExample(ctx, logger)
	if err != nil {
		log.Printf("C++ example failed: %v", err)
	}

	// Example 3: JavaScript array sum
	fmt.Println("\n=== Example 3: JavaScript Array Sum ===")
	err = runJavaScriptExample(ctx, logger)
	if err != nil {
		log.Printf("JavaScript example failed: %v", err)
	}
}

func runPythonFactorialExample(ctx context.Context, logger pipeline.Logger) error {
	// Create Docker execution step
	step := steps.NewDockerExecutionStep(logger)

	// Prepare test cases
	testCases := []types.TestCase{
		{
			ID:             "test_1",
			Input:          "5",
			ExpectedOutput: "120",
			Description:    "Calculate factorial of 5",
			TimeoutSeconds: 30,
		},
		{
			ID:             "test_2",
			Input:          "0",
			ExpectedOutput: "1",
			Description:    "Calculate factorial of 0",
			TimeoutSeconds: 30,
		},
		{
			ID:             "test_3",
			Input:          "10",
			ExpectedOutput: "3628800",
			Description:    "Calculate factorial of 10",
			TimeoutSeconds: 30,
		},
	}

	// Python solution code
	solutionCode := `def factorial(n):
    """Calculate factorial of n"""
    if n <= 1:
        return 1
    return n * factorial(n - 1)

def main():
    """Main function for testing"""
    import sys
    if len(sys.argv) > 1:
        n = int(sys.argv[1])
        print(factorial(n))
    else:
        # Interactive mode
        n = int(input())
        print(factorial(n))

if __name__ == "__main__":
    main()
`

	// Create execution data
	data := createExecutionData("python", solutionCode, testCases)

	// Execute
	return step.Execute(ctx, data)
}

func runCppCustomValidationExample(ctx context.Context, logger pipeline.Logger) error {
	step := steps.NewDockerExecutionStep(logger)

	// Test cases with custom validation
	testCases := []types.TestCase{
		{
			ID:                   "custom_test_1",
			CustomValidationCode: "CHECK(fibonacci(5) == 5); CHECK(fibonacci(10) == 55);",
			Description:          "Custom validation for fibonacci function",
			TimeoutSeconds:       30,
		},
		{
			ID:             "standard_test_1",
			Input:          "8",
			ExpectedOutput: "21",
			Description:    "Standard fibonacci test",
			TimeoutSeconds: 30,
		},
	}

	// C++ solution code
	solutionCode := `#include <iostream>

int fibonacci(int n) {
    if (n <= 1) return n;
    return fibonacci(n - 1) + fibonacci(n - 2);
}

int main() {
    int n;
    std::cin >> n;
    std::cout << fibonacci(n) << std::endl;
    return 0;
}
`

	data := createExecutionData("cpp", solutionCode, testCases)
	return step.Execute(ctx, data)
}

func runJavaScriptExample(ctx context.Context, logger pipeline.Logger) error {
	step := steps.NewDockerExecutionStep(logger)

	// JavaScript test cases
	testCases := []types.TestCase{
		{
			ID:             "test_1",
			Input:          "1 2 3 4 5",
			ExpectedOutput: "15",
			Description:    "Sum array [1,2,3,4,5]",
			TimeoutSeconds: 30,
		},
		{
			ID:             "test_2",
			Input:          "10 -5 3",
			ExpectedOutput: "8",
			Description:    "Sum array with negative numbers",
			TimeoutSeconds: 30,
		},
	}

	// JavaScript solution code
	solutionCode := `function sumArray(arr) {
    return arr.reduce((sum, num) => sum + num, 0);
}

function main() {
    const args = process.argv.slice(2);
    if (args.length > 0) {
        const numbers = args.map(Number);
        console.log(sumArray(numbers));
    } else {
        // Read from stdin
        const readline = require('readline');
        const rl = readline.createInterface({
            input: process.stdin,
            output: process.stdout
        });
        
        rl.on('line', (input) => {
            const numbers = input.split(' ').map(Number);
            console.log(sumArray(numbers));
            rl.close();
        });
    }
}

if (require.main === module) {
    main();
}

module.exports = { sumArray };
`

	data := createExecutionData("javascript", solutionCode, testCases)
	return step.Execute(ctx, data)
}

func createExecutionData(language, code string, testCases []types.TestCase) *pipeline.ExecutionData {
	// Create mock execution data
	data := &pipeline.ExecutionData{
		Code:        code,
		Language:    language,
		ChallengeID: "example_challenge",
		StudentID:   "student_123",
		SolutionID:  "solution_456",
		Config: &pipeline.ExecutionConfig{
			TimeoutSeconds:       30,
			MemoryLimitMB:        256,
			EnableNetwork:        false,
			EnvironmentVariables: make(map[string]string),
			DebugMode:            false,
		},
		Status:      pipeline.ExecutionStatusPending,
		Metadata:    make(map[string]string),
		TestResults: make([]*pipeline.TestResult, 0),
	}

	// Store test cases in metadata (this would normally be done by TestFetchingStep)
	testCasesJSON, _ := json.Marshal(testCases)
	data.Metadata["test_cases_json"] = string(testCasesJSON)

	return data
}
