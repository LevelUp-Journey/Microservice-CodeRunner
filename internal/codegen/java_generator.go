package codegen

import (
	"fmt"
	"strings"

	"code-runner/internal/types"
	"code-runner/internal/utils"
)

// JavaGenerator generates Java test code using JUnit
type JavaGenerator struct {
	*BaseGenerator
}

// NewJavaGenerator creates a new Java test generator
func NewJavaGenerator() TestGenerator {
	return &JavaGenerator{
		BaseGenerator: &BaseGenerator{
			language:      "java",
			fileExtension: ".java",
			executionCmd:  "mvn test",
		},
	}
}

// GenerateTestCode generates Java test code with JUnit framework
func (jg *JavaGenerator) GenerateTestCode(solution string, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error) {
	if functionInfo == nil {
		return "", fmt.Errorf("function information is required for Java code generation")
	}

	var testCode strings.Builder

	// Add package and imports
	testCode.WriteString("package com.levelup;\n\n")
	testCode.WriteString("import org.junit.jupiter.api.Test;\n")
	testCode.WriteString("import org.junit.jupiter.api.DisplayName;\n")
	testCode.WriteString("import static org.junit.jupiter.api.Assertions.*;\n\n")

	// Extract class name from solution or use default
	className := jg.extractClassName(solution)
	if className == "" {
		className = "Solution"
	}

	// Add solution code (extract class if needed)
	testCode.WriteString("// === SOLUTION CODE ===\n")
	testCode.WriteString(solution)
	testCode.WriteString("\n\n")

	// Generate test class
	testCode.WriteString(fmt.Sprintf("class %sTest {\n", className))

	// Generate test methods
	for i, testCase := range testCases {
		if !testCase.IsValid() {
			continue
		}

		if testCase.HasCustomValidation() {
			testCode.WriteString(jg.generateCustomTest(testCase, functionInfo, className, i+1))
		} else {
			testCode.WriteString(jg.generateStandardTest(testCase, functionInfo, className, i+1))
		}
	}

	testCode.WriteString("}\n")

	return testCode.String(), nil
}

// generateStandardTest generates a standard input/output test for Java
func (jg *JavaGenerator) generateStandardTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, className string, testNum int) string {
	var test strings.Builder

	methodName := fmt.Sprintf("test%s%s", strings.Title(functionInfo.Name), strings.Title(testCase.ID))
	test.WriteString(fmt.Sprintf("    @Test\n"))
	test.WriteString(fmt.Sprintf("    @DisplayName(\"%s: %s\")\n", testCase.ID, jg.escapeString(testCase.Description, "java")))
	test.WriteString(fmt.Sprintf("    void %s() {\n", methodName))

	// Create instance if needed
	test.WriteString(fmt.Sprintf("        %s solution = new %s();\n", className, className))

	// Parse input arguments
	inputArgs := jg.parseInputArguments(testCase.Input)
	expectedOutput := jg.formatTestOutput(testCase.ExpectedOutput)

	// Generate function call
	if len(inputArgs) > 0 {
		test.WriteString(fmt.Sprintf("        var result = solution.%s(%s);\n", functionInfo.Name, strings.Join(inputArgs, ", ")))
	} else {
		test.WriteString(fmt.Sprintf("        var result = solution.%s();\n", functionInfo.Name))
	}

	// Generate assertion
	if isNumeric(expectedOutput) {
		test.WriteString(fmt.Sprintf("        assertEquals(%s, result);\n", expectedOutput))
	} else {
		test.WriteString(fmt.Sprintf("        assertEquals(\"%s\", result);\n", jg.escapeString(expectedOutput, "java")))
	}

	test.WriteString("    }\n\n")
	return test.String()
}

// generateCustomTest generates a custom validation test for Java
func (jg *JavaGenerator) generateCustomTest(testCase types.TestCase, functionInfo *utils.FunctionInfo, className string, testNum int) string {
	var test strings.Builder

	methodName := fmt.Sprintf("testCustom%s", strings.Title(testCase.ID))
	test.WriteString(fmt.Sprintf("    @Test\n"))
	test.WriteString(fmt.Sprintf("    @DisplayName(\"Custom %s: %s\")\n", testCase.ID, jg.escapeString(testCase.Description, "java")))
	test.WriteString(fmt.Sprintf("    void %s() {\n", methodName))

	// Create instance
	test.WriteString(fmt.Sprintf("        %s solution = new %s();\n", className, className))

	// Add custom validation code with proper indentation
	customCode := strings.ReplaceAll(testCase.CustomValidationCode, "\n", "\n        ")
	test.WriteString("        " + customCode + "\n")

	test.WriteString("    }\n\n")
	return test.String()
}

// parseInputArguments parses input string into Java function arguments
func (jg *JavaGenerator) parseInputArguments(input string) []string {
	input = jg.formatTestInput(input)
	if input == "" {
		return []string{}
	}

	// Split by whitespace and format as Java arguments
	parts := strings.Fields(input)
	var args []string

	for _, part := range parts {
		if isNumeric(part) {
			args = append(args, part)
		} else {
			args = append(args, fmt.Sprintf("\"%s\"", jg.escapeString(part, "java")))
		}
	}

	return args
}

// extractClassName extracts class name from Java code
func (jg *JavaGenerator) extractClassName(code string) string {
	// Look for class declaration
	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "class ") {
			// Extract class name
			parts := strings.Fields(line)
			for i, part := range parts {
				if part == "class" && i+1 < len(parts) {
					className := parts[i+1]
					// Remove any generics or implements
					if idx := strings.Index(className, "<"); idx != -1 {
						className = className[:idx]
					}
					if idx := strings.Index(className, "{"); idx != -1 {
						className = className[:idx]
					}
					return className
				}
			}
		}
	}
	return ""
}
