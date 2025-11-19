package template

import (
	"fmt"
	"strings"

	"code-runner/internal/types"

	"github.com/google/uuid"
)

// TestGenerator handles generation of C++ test code
type TestGenerator struct {
	inputParser *InputParser
}

// NewTestGenerator creates a new test generator
func NewTestGenerator() *TestGenerator {
	return &TestGenerator{
		inputParser: NewInputParser(),
	}
}

// GenerateTestCode generates test code based on test cases
func (g *TestGenerator) GenerateTestCode(tests []*types.TestCase, functionName string, returnType string) (string, int) {
	var testLines []string
	testCount := 0

	// Detect if return type is char* or const char* (string pointers)
	isStringReturn := strings.Contains(returnType, "char") && strings.Contains(returnType, "*")

	for i, test := range tests {
		testCount++
		testID := test.CodeVersionTestID
		if testID == uuid.Nil {
			testID = uuid.New()
		}

		if test.HasCustomValidation() {
			// Use custom validation
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID.String()))
			testLines = append(testLines, test.CustomValidationCode)
			testLines = append(testLines, "}")
		} else {
			// Use standard input/output
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID.String()))
			// For simple cases, assume single input/output
			if test.Input != "" && test.ExpectedOutput != "" {
				// Parse input - generate setup code if there are arrays
				setupCode, functionCall := g.inputParser.ParseComplexInput(test.Input, functionName, i)
				expected := g.formatExpectedOutput(test.ExpectedOutput)

				if setupCode != "" {
					testLines = append(testLines, setupCode)
				}

				// If returns char*, use strcmp to compare strings
				if isStringReturn && strings.HasPrefix(expected, "\"") {
					testLines = append(testLines, fmt.Sprintf("    CHECK(strcmp(%s, %s) == 0);", functionCall, expected))
				} else {
					testLines = append(testLines, fmt.Sprintf("    CHECK(%s == %s);", functionCall, expected))
				}
			}
			testLines = append(testLines, "}")
		}
	}

	return strings.Join(testLines, "\n"), testCount
}


// formatExpectedOutput formats the expected output for C++
func (g *TestGenerator) formatExpectedOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}

	if g.isNumeric(output) {
		return output
	}

	if output == "true" || output == "false" {
		return output
	}

	if g.isQuotedString(output) {
		return output
	}

	return fmt.Sprintf("\"%s\"", output)
}



// isNumeric checks if a string represents a number
func (g *TestGenerator) isNumeric(s string) bool {
	if s == "" {
		return false
	}

	// Check if it's an integer or float
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

func (g *TestGenerator) isQuotedString(s string) bool {
	if len(s) < 2 {
		return false
	}

	first := s[0]
	last := s[len(s)-1]

	if first != last {
		return false
	}

	return first == '"' || first == '\''
}
