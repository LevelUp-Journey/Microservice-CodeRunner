package codegen

import (
	"fmt"
	"strings"

	"code-runner/internal/types"
	"code-runner/internal/utils"
)

// JavaScriptGenerator generates JavaScript test code using Jest
type JavaScriptGenerator struct {
	*BaseGenerator
}

// NewJavaScriptGenerator creates a new JavaScript test generator
func NewJavaScriptGenerator() TestGenerator {
	return &JavaScriptGenerator{
		BaseGenerator: &BaseGenerator{
			language:      "javascript",
			fileExtension: ".js",
			executionCmd:  "npm test",
		},
	}
}

// GenerateTestCode generates JavaScript test code with Jest framework
func (jsg *JavaScriptGenerator) GenerateTestCode(solution string, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error) {
	if functionInfo == nil {
		return "", fmt.Errorf("function information is required for JavaScript code generation")
	}

	var testCode strings.Builder

	// Add solution code
	testCode.WriteString("// === SOLUTION CODE ===\n")
	testCode.WriteString(solution)
	testCode.WriteString("\n\n")

	// Generate test suite
	testCode.WriteString(fmt.Sprintf("describe('%s Tests', () => {\n", functionInfo.Name))

	// Generate test cases
	for i, testCase := range testCases {
		if !testCase.IsValid() {
			continue
		}

		if testCase.HasCustomValidation() {
			testCode.WriteString(jsg.generateCustomTest(testCase, functionInfo, i+1))
		} else {
			testCode.WriteString(jsg.generateStandardTest(testCase, functionInfo, i+1))
		}
	}

	testCode.WriteString("});\n")

	return testCode.String(), nil
}

// generateStandardTest generates a standard input/output test for JavaScript
func (jsg *JavaScriptGenerator) generateStandardTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	testTitle := fmt.Sprintf("Test %d: %s", testNum, testCase.Description)
	test.WriteString(fmt.Sprintf("    test('%s', () => {\n", jsg.escapeString(testTitle, "javascript")))

	// Parse input arguments
	inputArgs := jsg.parseInputArguments(testCase.Input)
	expectedOutput := jsg.formatTestOutput(testCase.ExpectedOutput)

	// Generate function call
	if len(inputArgs) > 0 {
		test.WriteString(fmt.Sprintf("        const result = %s(%s);\n", functionInfo.Name, strings.Join(inputArgs, ", ")))
	} else {
		test.WriteString(fmt.Sprintf("        const result = %s();\n", functionInfo.Name))
	}

	// Generate assertion
	if isNumeric(expectedOutput) {
		test.WriteString(fmt.Sprintf("        expect(result).toBe(%s);\n", expectedOutput))
	} else {
		test.WriteString(fmt.Sprintf("        expect(result).toBe(\"%s\");\n", jsg.escapeString(expectedOutput, "javascript")))
	}

	test.WriteString("    });\n\n")
	return test.String()
}

// generateCustomTest generates a custom validation test for JavaScript
func (jsg *JavaScriptGenerator) generateCustomTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	testTitle := fmt.Sprintf("Custom Test %d: %s", testNum, testCase.Description)
	test.WriteString(fmt.Sprintf("    test('%s', () => {\n", jsg.escapeString(testTitle, "javascript")))

	// Add custom validation code with proper indentation
	customCode := strings.ReplaceAll(testCase.CustomValidationCode, "\n", "\n        ")
	test.WriteString("        " + customCode + "\n")

	test.WriteString("    });\n\n")
	return test.String()
}

// parseInputArguments parses input string into JavaScript function arguments
func (jsg *JavaScriptGenerator) parseInputArguments(input string) []string {
	input = jsg.formatTestInput(input)
	if input == "" {
		return []string{}
	}

	// Split by whitespace and format as JavaScript arguments
	parts := strings.Fields(input)
	var args []string

	for _, part := range parts {
		if isNumeric(part) {
			args = append(args, part)
		} else {
			args = append(args, fmt.Sprintf("\"%s\"", jsg.escapeString(part, "javascript")))
		}
	}

	return args
}