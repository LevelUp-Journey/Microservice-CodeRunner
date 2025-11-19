package template

import (
	"fmt"
	"regexp"
	"strings"
)

// InputParser handles parsing of complex C++ function inputs
type InputParser struct{}

// NewInputParser creates a new input parser
func NewInputParser() *InputParser {
	return &InputParser{}
}

// ParseComplexInput parses complex inputs that may include arrays and multiple parameters
func (p *InputParser) ParseComplexInput(input string, functionName string, testIndex int) (setupCode string, functionCall string) {
	input = strings.TrimSpace(input)

	// Detect if there are arrays in the input (format: [1,2,3] or similar)
	arrayRegex := regexp.MustCompile(`\[([^\]]+)\]`)

	if arrayRegex.MatchString(input) {
		// There is an array, we need to generate setup code
		var setupLines []string
		var args []string

		// Split by commas outside brackets
		parts := p.splitInputParameters(input)

		for _, part := range parts {
			part = strings.TrimSpace(part)

			if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
				// It's an array
				arrayContent := part[1 : len(part)-1] // Remove [ and ]
				arrayVarName := fmt.Sprintf("arr%d", testIndex)

				// Detect array type (numbers or strings)
				isStringArray := strings.Contains(arrayContent, "\"")

				if isStringArray {
					// String array: ["flower","flow","flight"]
					arraySize := p.countArrayElements(arrayContent)

					// Generate char* array declaration
					setupLines = append(setupLines, fmt.Sprintf("    char* %s[] = {%s};", arrayVarName, arrayContent))

					// Add array and size as arguments
					args = append(args, arrayVarName)
					args = append(args, fmt.Sprintf("%d", arraySize))
				} else {
					// Number array: [1,2,3]
					elements := strings.Split(arrayContent, ",")
					arraySize := len(elements)

					// Generate int array declaration
					setupLines = append(setupLines, fmt.Sprintf("    int %s[] = {%s};", arrayVarName, arrayContent))

					// Add array and size as arguments
					args = append(args, arrayVarName)
					args = append(args, fmt.Sprintf("%d", arraySize))
				}
			} else {
				// It's a simple parameter
				args = append(args, p.formatSimpleParameter(part))
			}
		}

		setupCode = strings.Join(setupLines, "\n")
		functionCall = fmt.Sprintf("%s(%s)", functionName, strings.Join(args, ", "))
		return setupCode, functionCall
	}

	// No arrays, use original formatInput but support multiple parameters
	parts := strings.Split(input, ",")
	if len(parts) > 1 {
		var formattedParts []string
		for _, part := range parts {
			formattedParts = append(formattedParts, p.formatSimpleParameter(strings.TrimSpace(part)))
		}
		functionCall = fmt.Sprintf("%s(%s)", functionName, strings.Join(formattedParts, ", "))
		return "", functionCall
	}

	// Simple input, single parameter
	functionCall = fmt.Sprintf("%s(%s)", functionName, p.formatSimpleParameter(input))
	return "", functionCall
}

// splitInputParameters splits input into parameters respecting array brackets
func (p *InputParser) splitInputParameters(input string) []string {
	var parts []string
	var current strings.Builder
	bracketDepth := 0

	for _, char := range input {
		if char == '[' {
			bracketDepth++
			current.WriteRune(char)
		} else if char == ']' {
			bracketDepth--
			current.WriteRune(char)
		} else if char == ',' && bracketDepth == 0 {
			// Comma outside brackets, it's a parameter separator
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteRune(char)
		}
	}

	// Add the last parameter
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// countArrayElements counts elements in an array respecting quotes
// Example: "flower","flow","flight" -> 3
func (p *InputParser) countArrayElements(arrayContent string) int {
	count := 0
	inQuotes := false

	for _, char := range arrayContent {
		if char == '"' {
			inQuotes = !inQuotes
			if !inQuotes {
				count++
			}
		}
	}

	return count
}

// formatSimpleParameter formats a simple parameter (not array)
func (p *InputParser) formatSimpleParameter(param string) string {
	param = strings.TrimSpace(param)

	if param == "" {
		return ""
	}

	if p.isNumeric(param) {
		return param
	}

	if param == "true" || param == "false" {
		return param
	}

	if p.isQuotedString(param) {
		return param
	}

	return fmt.Sprintf("\"%s\"", param)
}


// isNumeric checks if a string represents a number
func (p *InputParser) isNumeric(s string) bool {
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

func (p *InputParser) isQuotedString(s string) bool {
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
