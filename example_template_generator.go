package main

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"code-runner/internal/types"
)

// GeneratedTestCode represents the generated test code
type GeneratedTestCode struct {
	ID                  string
	Language            string
	GeneratorType       string
	TestCode            string
	ChallengeID         string
	TestCasesCount      int
	HasCustomValidation bool
	GenerationTimeMS    int64
	CodeSizeBytes       int
}

// Repository interface for storing generated test code
type GeneratedTestCodeRepository interface {
	Create(generatedTestCode *GeneratedTestCode) error
}

// Simple in-memory repository for demonstration
type MockGeneratedTestCodeRepository struct {
	stored []*GeneratedTestCode
}

func (r *MockGeneratedTestCodeRepository) Create(generatedTestCode *GeneratedTestCode) error {
	r.stored = append(r.stored, generatedTestCode)
	fmt.Printf("✅ Template stored in memory (ID: %s)\n", generatedTestCode.ID)
	return nil
}

// CppTemplateGenerator genera templates de C++ basados en ExecutionRequest
type CppTemplateGenerator struct {
	repo GeneratedTestCodeRepository
}

// NewCppTemplateGenerator crea una nueva instancia del generador
func NewCppTemplateGenerator(repo GeneratedTestCodeRepository) *CppTemplateGenerator {
	return &CppTemplateGenerator{repo: repo}
}

// GenerateTemplate crea el template completo y lo guarda
func (g *CppTemplateGenerator) GenerateTemplate(req *types.ExecutionRequest) (*GeneratedTestCode, error) {
	startTime := time.Now()

	// Extraer nombre de función usando regex
	functionName, err := g.extractFunctionName(req.Code)
	if err != nil {
		return nil, fmt.Errorf("error extracting function name: %w", err)
	}

	// Generar código de tests
	testCode, testCount := g.generateTestCode(req.TestCases, functionName)

	// Crear template completo
	template := g.buildTemplate(req.Code, testCode)

	// Crear registro
	record := &GeneratedTestCode{
		ID:                  fmt.Sprintf("template_%d", time.Now().Unix()),
		Language:            req.Language,
		GeneratorType:       "cpp_template",
		TestCode:            template,
		ChallengeID:         req.ChallengeID,
		TestCasesCount:      testCount,
		HasCustomValidation: g.hasCustomValidation(req.TestCases),
		GenerationTimeMS:    time.Since(startTime).Milliseconds(),
		CodeSizeBytes:       len(template),
	}

	// Guardar
	if err := g.repo.Create(record); err != nil {
		return nil, fmt.Errorf("error saving template: %w", err)
	}

	return record, nil
}

// extractFunctionName usa regex para encontrar el nombre de la función principal
func (g *CppTemplateGenerator) extractFunctionName(code string) (string, error) {
	re := regexp.MustCompile(`(?m)^\s*(?:int|void|double|float|char|string|bool)\s+(\w+)\s*\([^)]*\)\s*\{`)
	matches := re.FindStringSubmatch(code)

	if len(matches) < 2 {
		return "", fmt.Errorf("no se encontró una función válida en el código")
	}

	return matches[1], nil
}

// generateTestCode genera el código de tests basado en los test cases
func (g *CppTemplateGenerator) generateTestCode(tests []*types.TestCase, functionName string) (string, int) {
	var testLines []string
	testCount := 0

	for _, test := range tests {
		testCount++
		testID := test.TestID
		if testID == "" {
			testID = fmt.Sprintf("test_%d", testCount)
		}

		if test.HasCustomValidation() {
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID))
			testLines = append(testLines, test.CustomValidationCode)
			testLines = append(testLines, "}")
		} else {
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID))
			if test.Input != "" && test.ExpectedOutput != "" {
				input := g.formatInput(test.Input)
				expected := g.formatExpectedOutput(test.ExpectedOutput)
				testLines = append(testLines, fmt.Sprintf("    CHECK(%s(%s) == %s);", functionName, input, expected))
			}
			testLines = append(testLines, "}")
		}
	}

	return strings.Join(testLines, "\n"), testCount
}

// formatInput formatea el input para C++
func (g *CppTemplateGenerator) formatInput(input string) string {
	input = strings.TrimSpace(input)
	if g.isNumeric(input) {
		return input
	}
	return fmt.Sprintf("\"%s\"", input)
}

// formatExpectedOutput formatea el output esperado para C++
func (g *CppTemplateGenerator) formatExpectedOutput(output string) string {
	output = strings.TrimSpace(output)
	if g.isNumeric(output) {
		return output
	}
	return fmt.Sprintf("\"%s\"", output)
}

// isNumeric verifica si una cadena representa un número
func (g *CppTemplateGenerator) isNumeric(s string) bool {
	if s == "" {
		return false
	}
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

// hasCustomValidation verifica si algún test tiene validación personalizada
func (g *CppTemplateGenerator) hasCustomValidation(tests []*types.TestCase) bool {
	for _, test := range tests {
		if test.HasCustomValidation() {
			return true
		}
	}
	return false
}

// buildTemplate construye el template completo
func (g *CppTemplateGenerator) buildTemplate(solutionCode, testCode string) string {
	template := `// Start Test
#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN
// Solution - Start
%s
// Solution - End

// Tests - Start
%s
// Tests - End`

	return fmt.Sprintf(template, solutionCode, testCode)
}

func main() {
	fmt.Println("🚀 C++ Template Generator Example")
	fmt.Println("==================================")

	// Create mock repository
	repo := &MockGeneratedTestCodeRepository{}

	// Create template generator
	generator := NewCppTemplateGenerator(repo)

	// Example ExecutionRequest (simulating proto data)
	req := &types.ExecutionRequest{
		SolutionID:  "solution_123",
		ChallengeID: "challenge_456",
		StudentID:   "student_789",
		Language:    "c_plus_plus",
		Code: `int fibonacci(int n) {
    if (n < 0) throw std::invalid_argument("n debe ser >= 0");
    if (n == 0) return 0;
    if (n == 1) return 1;
    return fibonacci(n - 1) + fibonacci(n - 2);
}`,
		TestCases: []*types.TestCase{
			{
				TestID:         "test_1",
				Input:          "0",
				ExpectedOutput: "0",
			},
			{
				TestID:         "test_2",
				Input:          "1",
				ExpectedOutput: "1",
			},
			{
				TestID:         "test_3",
				Input:          "5",
				ExpectedOutput: "5",
			},
			{
				TestID: "test_custom",
				CustomValidationCode: `    // Custom validation for fibonacci
    CHECK(fibonacci(2) == 1);
    CHECK(fibonacci(3) == 2);
    CHECK(fibonacci(10) == 55);`,
			},
		},
	}

	// Generate template
	fmt.Println("🔧 Generating C++ template...")
	generatedTemplate, err := generator.GenerateTemplate(req)
	if err != nil {
		log.Fatalf("Failed to generate template: %v", err)
	}

	fmt.Printf("✅ Template generated successfully!\n")
	fmt.Printf("🏷️  Language: %s\n", generatedTemplate.Language)
	fmt.Printf("🔢 Test Cases: %d\n", generatedTemplate.TestCasesCount)
	fmt.Printf("⏱️  Generation Time: %d ms\n", generatedTemplate.GenerationTimeMS)
	fmt.Printf("📏 Code Size: %d bytes\n", generatedTemplate.CodeSizeBytes)
	fmt.Printf("🔧 Generator: %s\n", generatedTemplate.GeneratorType)
	fmt.Printf("🎯 Has Custom Validation: %v\n", generatedTemplate.HasCustomValidation)

	fmt.Println("\n📄 Generated Template:")
	fmt.Println("========================================")
	fmt.Println(generatedTemplate.TestCode)
	fmt.Println("========================================")

	fmt.Printf("\n📊 Total templates stored: %d\n", len(repo.stored))
}
