package codegen

import (
	"fmt"
	"strings"

	"code-runner/internal/types"
	"code-runner/internal/utils"
)

// GoGenerator generates Go test code using native testing framework
type GoGenerator struct {
	*BaseGenerator
}

// NewGoGenerator creates a new Go test generator
func NewGoGenerator() TestGenerator {
	return &GoGenerator{
		BaseGenerator: &BaseGenerator{
			language:      "go",
			fileExtension: ".go",
			executionCmd:  "go test -v -json",
		},
	}
}

// GenerateTestCode generates Go test code with native testing framework
func (gg *GoGenerator) GenerateTestCode(solution string, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error) {
	if functionInfo == nil {
		return "", fmt.Errorf("function information is required for Go code generation")
	}

	var testCode strings.Builder

	// Add package and imports
	testCode.WriteString("package main\n\n")
	testCode.WriteString("import (\n")
	testCode.WriteString("    \"testing\"\n")
	testCode.WriteString("    \"reflect\"\n")
	testCode.WriteString(")\n\n")

	// Add solution code
	testCode.WriteString("// === SOLUTION CODE ===\n")
	testCode.WriteString(solution)
	testCode.WriteString("\n\n")

	// Generate test functions
	for i, testCase := range testCases {
		if !testCase.IsValid() {
			continue
		}

		if testCase.HasCustomValidation() {
			testCode.WriteString(gg.generateCustomTest(testCase, functionInfo, i+1))
		} else {
			testCode.WriteString(gg.generateStandardTest(testCase, functionInfo, i+1))
		}
	}

	return testCode.String(), nil
}

// generateStandardTest generates a standard input/output test for Go
func (gg *GoGenerator) generateStandardTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	testName := fmt.Sprintf("Test%s%s", strings.Title(functionInfo.Name), strings.Title(testCase.ID))
	test.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testName))
	test.WriteString(fmt.Sprintf("    // %s: %s\n", testCase.ID, testCase.Description))

	// Parse input arguments
	inputArgs := gg.parseInputArguments(testCase.Input)
	expectedOutput := gg.formatTestOutput(testCase.ExpectedOutput)

	// Generate function call
	var resultCall string
	if len(inputArgs) > 0 {
		resultCall = fmt.Sprintf("%s(%s)", functionInfo.Name, strings.Join(inputArgs, ", "))
	} else {
		resultCall = fmt.Sprintf("%s()", functionInfo.Name)
	}

	// Generate assertion based on expected output type
	if isNumeric(expectedOutput) {
		test.WriteString(fmt.Sprintf("    result := %s\n", resultCall))
		test.WriteString(fmt.Sprintf("    expected := %s\n", expectedOutput))
		test.WriteString("    if result != expected {\n")
		test.WriteString("        t.Errorf(\"Expected %v, got %v\", expected, result)\n")
		test.WriteString("    }\n")
	} else {
		test.WriteString(fmt.Sprintf("    result := %s\n", resultCall))
		test.WriteString(fmt.Sprintf("    expected := \"%s\"\n", gg.escapeString(expectedOutput, "go")))
		test.WriteString("    if result != expected {\n")
		test.WriteString("        t.Errorf(\"Expected %s, got %s\", expected, result)\n")
		test.WriteString("    }\n")
	}

	test.WriteString("}\n\n")
	return test.String()
}

// generateCustomTest generates a custom validation test for Go
func (gg *GoGenerator) generateCustomTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	testName := fmt.Sprintf("TestCustom%s", strings.Title(testCase.ID))
	test.WriteString(fmt.Sprintf("func %s(t *testing.T) {\n", testName))
	test.WriteString(fmt.Sprintf("    // Custom %s: %s\n", testCase.ID, testCase.Description))

	// Add custom validation code with proper indentation
	customCode := strings.ReplaceAll(testCase.CustomValidationCode, "\n", "\n    ")
	test.WriteString("    " + customCode + "\n")

	test.WriteString("}\n\n")
	return test.String()
}

// parseInputArguments parses input string into Go function arguments
func (gg *GoGenerator) parseInputArguments(input string) []string {
	input = gg.formatTestInput(input)
	if input == "" {
		return []string{}
	}

	// Split by whitespace and format as Go arguments
	parts := strings.Fields(input)
	var args []string

	for _, part := range parts {
		if isNumeric(part) {
			args = append(args, part)
		} else {
			args = append(args, fmt.Sprintf("\"%s\"", gg.escapeString(part, "go")))
		}
	}

	return args
}
