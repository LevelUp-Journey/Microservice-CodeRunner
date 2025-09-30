package codegen

import (
	"fmt"
	"strings"

	"code-runner/internal/types"
	"code-runner/internal/utils"
)

// CppGenerator generates C++ test code using doctest
type CppGenerator struct {
	*BaseGenerator
}

// NewCppGenerator creates a new C++ test generator
func NewCppGenerator() TestGenerator {
	return &CppGenerator{
		BaseGenerator: &BaseGenerator{
			language:      "c_plus_plus",
			fileExtension: ".cpp",
			executionCmd:  "g++ -std=c++17 -I/usr/local/include main.cpp -o test_runner && ./test_runner",
		},
	}
}

// GenerateTestCode generates C++ test code with doctest framework
func (cg *CppGenerator) GenerateTestCode(solution string, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error) {
	if functionInfo == nil {
		return "", fmt.Errorf("function information is required for C++ code generation")
	}

	var testCode strings.Builder

	// Add doctest header and solution code
	testCode.WriteString("#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN\n")
	testCode.WriteString("#include <doctest.h>\n")
	testCode.WriteString("#include <iostream>\n")
	testCode.WriteString("#include <string>\n")
	testCode.WriteString("#include <sstream>\n\n")

	// Add solution code
	testCode.WriteString("// === SOLUTION CODE ===\n")
	testCode.WriteString(solution)
	testCode.WriteString("\n\n")

	// Generate test cases
	testCode.WriteString("// === TEST CASES ===\n")
	for i, testCase := range testCases {
		if !testCase.IsValid() {
			continue
		}

		if testCase.HasCustomValidation() {
			// Custom validation code
			testCode.WriteString(fmt.Sprintf("TEST_CASE(\"Custom Test %d: %s\") {\n", i+1, cg.escapeString(testCase.Description, "c_plus_plus")))
			testCode.WriteString("    // Custom validation code\n")
			testCode.WriteString("    " + testCase.CustomValidationCode + "\n")
			testCode.WriteString("}\n\n")
		} else {
			// Standard input/output test
			testCode.WriteString(cg.generateStandardTest(testCase, functionInfo, i+1))
		}
	}

	return testCode.String(), nil
}

// generateStandardTest generates a standard input/output test for C++
func (cg *CppGenerator) generateStandardTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, testNum int) string {
	var test strings.Builder

	test.WriteString(fmt.Sprintf("TEST_CASE(\"%s: %s\") {\n", testCase.ID, cg.escapeString(testCase.Description, "c_plus_plus")))

	// Parse input parameters
	inputArgs := cg.parseInputArguments(testCase.Input, functionInfo)
	expectedOutput := cg.formatTestOutput(testCase.ExpectedOutput)

	// Generate function call
	if len(inputArgs) > 0 {
		test.WriteString(fmt.Sprintf("    auto result = %s(%s);\n", functionInfo.Name, strings.Join(inputArgs, ", ")))
	} else {
		test.WriteString(fmt.Sprintf("    auto result = %s();\n", functionInfo.Name))
	}

	// Generate assertion based on expected output type
	if isNumeric(expectedOutput) {
		test.WriteString(fmt.Sprintf("    CHECK(result == %s);\n", expectedOutput))
	} else {
		test.WriteString(fmt.Sprintf("    CHECK(result == \"%s\");\n", cg.escapeString(expectedOutput, "c_plus_plus")))
	}

	test.WriteString("}\n\n")

	return test.String()
}

// parseInputArguments parses input string into C++ function arguments
func (cg *CppGenerator) parseInputArguments(input string, functionInfo *utils.FunctionInfo) []string {
	input = cg.formatTestInput(input)
	if input == "" {
		return []string{}
	}

	// Split by whitespace and format as C++ arguments
	parts := strings.Fields(input)
	var args []string

	for _, part := range parts {
		if isNumeric(part) {
			args = append(args, part)
		} else {
			args = append(args, fmt.Sprintf("\"%s\"", cg.escapeString(part, "c_plus_plus")))
		}
	}

	return args
}

// isNumeric checks if a string represents a numeric value
func isNumeric(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}

	// Check for integer or float
	for i, char := range s {
		if i == 0 && (char == '-' || char == '+') {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		if char == '.' {
			continue
		}
		return false
	}
	return true
}
