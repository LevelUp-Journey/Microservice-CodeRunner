package utils

import (
	"fmt"
	"regexp"
	"strings"
)

// FunctionInfo contains parsed function information
type FunctionInfo struct {
	Name       string
	Parameters []string
	ReturnType string
	Language   string
	FullMatch  string
}

// FunctionParser provides language-specific function parsing
type FunctionParser struct {
	patterns map[string]*regexp.Regexp
}

// NewFunctionParser creates a new function parser with language patterns
func NewFunctionParser() *FunctionParser {
	return &FunctionParser{
		patterns: map[string]*regexp.Regexp{
			// C++ function patterns (supports various formats)
			"cpp": regexp.MustCompile(`(?m)^\s*(?:(?:inline|static|virtual|extern)\s+)*(?:[\w:]+\s+)*(\w+)\s*\([^)]*\)\s*(?:const)?\s*(?:\{|;)`),

			// Python function patterns (def and async def)
			"python": regexp.MustCompile(`(?m)^\s*(?:async\s+)?def\s+(\w+)\s*\([^)]*\)\s*(?:->\s*[\w\[\], ]+)?\s*:`),

			// JavaScript function patterns (function, arrow functions, methods)
			"javascript": regexp.MustCompile(`(?m)(?:^\s*(?:export\s+)?(?:async\s+)?function\s+(\w+)|^\s*(?:const|let|var)\s+(\w+)\s*=\s*(?:async\s+)?(?:\([^)]*\)\s*=>|\([^)]*\)\s*=>\s*\{)|^\s*(\w+)\s*:\s*(?:async\s+)?(?:function\s*\([^)]*\)|\([^)]*\)\s*=>))`),

			// Java method patterns (public, private, protected, static methods)
			"java": regexp.MustCompile(`(?m)^\s*(?:public|private|protected)?\s*(?:static)?\s*(?:final)?\s*(?:\w+(?:<[^>]*>)?\s+)+(\w+)\s*\([^)]*\)\s*(?:throws\s+[\w\s,]+)?\s*\{`),

			// Go function patterns (func declarations)
			"go": regexp.MustCompile(`(?m)^\s*func\s+(?:\([^)]*\)\s+)?(\w+)\s*\([^)]*\)(?:\s*[\w\[\]]*)?(?:\s*\{|$)`),
		},
	}
}

// ParseFunctions extracts function names from code for a specific language
func (fp *FunctionParser) ParseFunctions(code, language string) ([]FunctionInfo, error) {
	pattern, exists := fp.patterns[strings.ToLower(language)]
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}

	matches := pattern.FindAllStringSubmatch(code, -1)
	if len(matches) == 0 {
		return []FunctionInfo{}, nil
	}

	var functions []FunctionInfo
	seen := make(map[string]bool) // Avoid duplicates

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		// Find the first non-empty capture group (function name)
		var functionName string
		for i := 1; i < len(match); i++ {
			if match[i] != "" {
				functionName = match[i]
				break
			}
		}

		if functionName == "" || seen[functionName] {
			continue
		}

		seen[functionName] = true

		// Extract additional info based on language
		function := FunctionInfo{
			Name:      functionName,
			Language:  language,
			FullMatch: match[0],
		}

		// Language-specific parameter extraction
		switch strings.ToLower(language) {
		case "cpp", "java":
			function.Parameters = fp.extractParametersFromMatch(match[0], language)
		case "python":
			function.Parameters = fp.extractPythonParameters(match[0])
		case "javascript":
			function.Parameters = fp.extractJSParameters(match[0])
		case "go":
			function.Parameters = fp.extractGoParameters(match[0])
		}

		functions = append(functions, function)
	}

	return functions, nil
}

// extractParametersFromMatch extracts parameters from the full match string
func (fp *FunctionParser) extractParametersFromMatch(fullMatch, language string) []string {
	// Extract parameters between parentheses
	paramPattern := regexp.MustCompile(`\(([^)]*)\)`)
	paramMatch := paramPattern.FindStringSubmatch(fullMatch)
	if len(paramMatch) < 2 {
		return []string{}
	}

	paramStr := strings.TrimSpace(paramMatch[1])
	if paramStr == "" {
		return []string{}
	}

	// Split by comma and clean up
	params := strings.Split(paramStr, ",")
	var cleanParams []string
	for _, param := range params {
		param = strings.TrimSpace(param)
		if param != "" {
			cleanParams = append(cleanParams, param)
		}
	}

	return cleanParams
}

// extractPythonParameters extracts Python function parameters
func (fp *FunctionParser) extractPythonParameters(fullMatch string) []string {
	paramPattern := regexp.MustCompile(`\(([^)]*)\)`)
	paramMatch := paramPattern.FindStringSubmatch(fullMatch)
	if len(paramMatch) < 2 {
		return []string{}
	}

	paramStr := strings.TrimSpace(paramMatch[1])
	if paramStr == "" {
		return []string{}
	}

	// Python-specific parameter parsing (handle defaults, *args, **kwargs)
	params := strings.Split(paramStr, ",")
	var cleanParams []string
	for _, param := range params {
		param = strings.TrimSpace(param)
		if param != "" && param != "self" && param != "cls" {
			// Extract just parameter name (remove type hints and defaults)
			if colonIndex := strings.Index(param, ":"); colonIndex != -1 {
				param = strings.TrimSpace(param[:colonIndex])
			}
			if equalIndex := strings.Index(param, "="); equalIndex != -1 {
				param = strings.TrimSpace(param[:equalIndex])
			}
			cleanParams = append(cleanParams, param)
		}
	}

	return cleanParams
}

// extractJSParameters extracts JavaScript function parameters
func (fp *FunctionParser) extractJSParameters(fullMatch string) []string {
	// Handle different JS function formats
	paramPattern := regexp.MustCompile(`\(([^)]*)\)`)
	paramMatch := paramPattern.FindStringSubmatch(fullMatch)
	if len(paramMatch) < 2 {
		return []string{}
	}

	return fp.splitParameters(paramMatch[1])
}

// extractGoParameters extracts Go function parameters
func (fp *FunctionParser) extractGoParameters(fullMatch string) []string {
	paramPattern := regexp.MustCompile(`\(([^)]*)\)`)
	paramMatch := paramPattern.FindStringSubmatch(fullMatch)
	if len(paramMatch) < 2 {
		return []string{}
	}

	return fp.splitParameters(paramMatch[1])
}

// splitParameters splits parameter string by comma and cleans up
func (fp *FunctionParser) splitParameters(paramStr string) []string {
	paramStr = strings.TrimSpace(paramStr)
	if paramStr == "" {
		return []string{}
	}

	params := strings.Split(paramStr, ",")
	var cleanParams []string
	for _, param := range params {
		param = strings.TrimSpace(param)
		if param != "" {
			cleanParams = append(cleanParams, param)
		}
	}

	return cleanParams
}

// GetMainFunction attempts to identify the main/entry function in code
func (fp *FunctionParser) GetMainFunction(code, language string) (*FunctionInfo, error) {
	functions, err := fp.ParseFunctions(code, language)
	if err != nil {
		return nil, err
	}

	// Language-specific main function patterns
	mainPatterns := map[string][]string{
		"cpp":        {"main"},
		"python":     {"main", "solve", "solution"},
		"javascript": {"main", "solve", "solution", "default"},
		"java":       {"main"},
		"go":         {"main", "Main"},
	}

	patterns, exists := mainPatterns[strings.ToLower(language)]
	if !exists {
		if len(functions) > 0 {
			return &functions[0], nil // Return first function found
		}
		return nil, fmt.Errorf("no functions found in %s code", language)
	}

	// Look for main function patterns
	for _, pattern := range patterns {
		for _, function := range functions {
			if strings.EqualFold(function.Name, pattern) {
				return &function, nil
			}
		}
	}

	// If no main function found, return the first function
	if len(functions) > 0 {
		return &functions[0], nil
	}

	return nil, fmt.Errorf("no functions found in %s code", language)
}
