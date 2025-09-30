package codegen

import (
	"fmt"
	"strings"

	"code-runner/internal/types"
	"code-runner/internal/utils"
)

// PythonGenerator generates Python test code using pytest
type PythonGenerator struct {
	*BaseGenerator
}

// NewPythonGenerator creates a new Python test generator
func NewPythonGenerator() TestGenerator {
	return &PythonGenerator{
		BaseGenerator: &BaseGenerator{
			language:      "python",
			fileExtension: ".py",
			executionCmd:  "python -m pytest test_main.py -v --json-report --json-report-file=test-results.json",
		},
	}
}

// GenerateTestCode generates Python test code with pytest framework
func (pg *PythonGenerator) GenerateTestCode(solution string, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error) {
	if functionInfo == nil {
		return "", fmt.Errorf("function information is required for Python code generation")
	}

	var testCode strings.Builder

	// Add imports and solution code
	testCode.WriteString("import pytest\n")
	testCode.WriteString("import sys\n")
	testCode.WriteString("import os\n")
	testCode.WriteString("from io import StringIO\n\n")

	testCode.WriteString("# === SOLUTION CODE ===\n")
	testCode.WriteString(solution)
	testCode.WriteString("\n\n")

	// Generate test class
	testCode.WriteString("class TestSolution:\n")
	testCode.WriteString("    \"\"\"Test class for solution validation\"\"\"\n\n")

	// Generate test methods
	for i, testCase := range testCases {
		if !testCase.IsValid() {
			continue
		}

		if testCase.HasCustomValidation() {
			testCode.WriteString(pg.generateCustomTest(testCase, functionInfo, i+1))
		} else {
			testCode.WriteString(pg.generateStandardTest(testCase, functionInfo, i+1))
		}
	}

	return testCode.String(), nil
}

// generateStandardTest generates a standard input/output test for Python
func (pg *PythonGenerator) generateStandardTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	methodName := fmt.Sprintf("test_%s_%s", strings.ToLower(functionInfo.Name), testCase.ID)
	test.WriteString(fmt.Sprintf("    def %s(self):\n", methodName))
	test.WriteString(fmt.Sprintf("        \"\"\"%s\"\"\"\n", pg.escapeString(testCase.Description, "python")))

	// Parse input arguments
	inputArgs := pg.parseInputArguments(testCase.Input)
	expectedOutput := pg.formatTestOutput(testCase.ExpectedOutput)

	// Generate function call
	if len(inputArgs) > 0 {
		test.WriteString(fmt.Sprintf("        result = %s(%s)\n", functionInfo.Name, strings.Join(inputArgs, ", ")))
	} else {
		test.WriteString(fmt.Sprintf("        result = %s()\n", functionInfo.Name))
	}

	// Generate assertion
	if isNumeric(expectedOutput) {
		test.WriteString(fmt.Sprintf("        assert result == %s\n", expectedOutput))
	} else {
		test.WriteString(fmt.Sprintf("        assert result == \"%s\"\n", pg.escapeString(expectedOutput, "python")))
	}

	test.WriteString("\n")
	return test.String()
}

// generateCustomTest generates a custom validation test for Python
func (pg *PythonGenerator) generateCustomTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	methodName := fmt.Sprintf("test_custom_%s", testCase.ID)
	test.WriteString(fmt.Sprintf("    def %s(self):\n", methodName))
	test.WriteString(fmt.Sprintf("        \"\"\"%s\"\"\"\n", pg.escapeString(testCase.Description, "python")))

	// Add custom validation code with proper indentation
	customCode := strings.ReplaceAll(testCase.CustomValidationCode, "\n", "\n        ")
	test.WriteString("        " + customCode + "\n")

	test.WriteString("\n")
	return test.String()
}

// parseInputArguments parses input string into Python function arguments
func (pg *PythonGenerator) parseInputArguments(input string) []string {
	input = pg.formatTestInput(input)
	if input == "" {
		return []string{}
	}

	// Split by whitespace and format as Python arguments
	parts := strings.Fields(input)
	var args []string

	for _, part := range parts {
		if isNumeric(part) {
			args = append(args, part)
		} else {
			args = append(args, fmt.Sprintf("\"%s\"", pg.escapeString(part, "python")))
		}
	}

	return args
}
