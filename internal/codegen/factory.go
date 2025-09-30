package codegen

import (
	"fmt"
	"strings"

	"code-runner/internal/types"
	"code-runner/internal/utils"
)

// TestGenerator interface for language-specific test generation
type TestGenerator interface {
	GenerateTestCode(solution string, testCases []types.TestCase, functionInfo *utils.FunctionInfo) (string, error)
	GetLanguage() string
	GetFileExtension() string
	GetExecutionCommand() string
}

// GeneratorFactory creates test generators for different languages
type GeneratorFactory struct {
	generators map[string]TestGenerator
}

// NewGeneratorFactory creates a new generator factory
func NewGeneratorFactory() *GeneratorFactory {
	factory := &GeneratorFactory{
		generators: make(map[string]TestGenerator),
	}

	// Register language generators
	factory.RegisterGenerator(NewCppGenerator())
	factory.RegisterGenerator(NewPythonGenerator())
	factory.RegisterGenerator(NewJavaScriptGenerator())
	factory.RegisterGenerator(NewJavaGenerator())
	factory.RegisterGenerator(NewGoGenerator())

	return factory
}

// RegisterGenerator registers a new test generator
func (gf *GeneratorFactory) RegisterGenerator(generator TestGenerator) {
	gf.generators[strings.ToLower(generator.GetLanguage())] = generator
}

// GetGenerator returns a test generator for the specified language
func (gf *GeneratorFactory) GetGenerator(language string) (TestGenerator, error) {
	generator, exists := gf.generators[strings.ToLower(language)]
	if !exists {
		return nil, fmt.Errorf("unsupported language: %s", language)
	}
	return generator, nil
}

// GetSupportedLanguages returns list of supported languages
func (gf *GeneratorFactory) GetSupportedLanguages() []string {
	var languages []string
	for lang := range gf.generators {
		languages = append(languages, lang)
	}
	return languages
}

// BaseGenerator provides common functionality for all generators
type BaseGenerator struct {
	language        string
	fileExtension   string
	executionCmd    string
}

// GetLanguage returns the language name
func (bg *BaseGenerator) GetLanguage() string {
	return bg.language
}

// GetFileExtension returns the file extension for this language
func (bg *BaseGenerator) GetFileExtension() string {
	return bg.fileExtension
}

// GetExecutionCommand returns the command to execute tests
func (bg *BaseGenerator) GetExecutionCommand() string {
	return bg.executionCmd
}

// formatTestInput formats input for test cases
func (bg *BaseGenerator) formatTestInput(input string) string {
	// Remove extra whitespace and normalize line endings
	return strings.TrimSpace(strings.ReplaceAll(input, "\r\n", "\n"))
}

// formatTestOutput formats expected output for test cases
func (bg *BaseGenerator) formatTestOutput(output string) string {
	// Remove extra whitespace and normalize line endings
	return strings.TrimSpace(strings.ReplaceAll(output, "\r\n", "\n"))
}

// escapeString escapes special characters for the target language
func (bg *BaseGenerator) escapeString(str, language string) string {
	// Basic escaping - can be extended per language
	str = strings.ReplaceAll(str, "\\", "\\\\")
	str = strings.ReplaceAll(str, "\"", "\\\"")
	str = strings.ReplaceAll(str, "\n", "\\n")
	str = strings.ReplaceAll(str, "\t", "\\t")
	return str
}