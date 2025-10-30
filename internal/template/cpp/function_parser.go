package template

import (
	"fmt"
	"regexp"
	"strings"
)

// FunctionParser handles extraction of function information from C++ code
type FunctionParser struct{}

// NewFunctionParser creates a new function parser
func NewFunctionParser() *FunctionParser {
	return &FunctionParser{}
}

// ExtractFunctionInfo extracts the function name and return type from C++ code
func (p *FunctionParser) ExtractFunctionInfo(code string) (string, string, error) {
	// Regex improved to capture return type and function name
	// Supports:
	//   - Basic types: int, char, double, float, etc.
	//   - stdint types: int64_t, uint32_t, size_t, etc.
	//   - STL types: vector<T>, std::string, etc.
	//   - Modifiers: const, unsigned, *, &
	// Group 1: complete type
	// Group 2: function name
	re := regexp.MustCompile(`(?m)^\s*((?:const\s+)?(?:unsigned\s+)?(?:int|void|double|float|char|string|bool|auto|long|short|size_t|int8_t|int16_t|int32_t|int64_t|uint8_t|uint16_t|uint32_t|uint64_t|vector<[^>]+>|std::string|std::vector<[^>]+>)(?:\s*\*|\s*&)?)\s+(\w+)\s*\([^)]*\)\s*\{`)
	matches := re.FindStringSubmatch(code)

	if len(matches) < 3 {
		return "", "", fmt.Errorf("no valid function found in code")
	}

	returnType := strings.TrimSpace(matches[1])
	functionName := matches[2]

	return functionName, returnType, nil
}

// ExtractFunctionName extracts just the function name (backward compatibility)
func (p *FunctionParser) ExtractFunctionName(code string) (string, error) {
	name, _, err := p.ExtractFunctionInfo(code)
	return name, err
}
