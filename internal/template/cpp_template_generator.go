package template

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"code-runner/internal/database/models"
	"code-runner/internal/database/repository"
	"code-runner/internal/types"

	"github.com/google/uuid"
)

// CppTemplateGenerator genera templates de C++ basados en ExecutionRequest
type CppTemplateGenerator struct {
	repo *repository.GeneratedTestCodeRepository
}

// NewCppTemplateGenerator crea una nueva instancia del generador
func NewCppTemplateGenerator(repo *repository.GeneratedTestCodeRepository) *CppTemplateGenerator {
	return &CppTemplateGenerator{repo: repo}
}

// GenerateTemplate crea el template completo y lo guarda en la base de datos
func (g *CppTemplateGenerator) GenerateTemplate(req *types.ExecutionRequest, executionID uuid.UUID) (*models.GeneratedTestCode, error) {
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

	// Crear registro para base de datos
	record := &models.GeneratedTestCode{
		ExecutionID:         executionID, // Usar el ExecutionID pasado como parámetro
		Language:            req.Language,
		GeneratorType:       "cpp_template",
		TestCode:            template,
		ChallengeID:         req.CodeVersionID, // Usando CodeVersionID como ChallengeID
		TestCasesCount:      testCount,
		HasCustomValidation: g.hasCustomValidation(req.TestCases),
		GenerationTimeMS:    time.Since(startTime).Milliseconds(),
		CodeSizeBytes:       len(template),
	}

	// Guardar en base de datos
	if err := g.repo.Create(record); err != nil {
		return nil, fmt.Errorf("error saving template to database: %w", err)
	}

	return record, nil
}

// extractFunctionName usa regex para encontrar el nombre de la función principal
func (g *CppTemplateGenerator) extractFunctionName(code string) (string, error) {
	// Regex para encontrar funciones en C++: tipo_retorno nombre_funcion(parametros)
	// Captura el nombre de la función (grupo 2)
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
		testID := test.CodeVersionTestID
		if testID == "" {
			testID = fmt.Sprintf("test_%d", testCount)
		}

		if test.HasCustomValidation() {
			// Usar validación personalizada
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID))
			testLines = append(testLines, test.CustomValidationCode)
			testLines = append(testLines, "}")
		} else {
			// Usar input/output estándar
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID))
			// Para casos simples, asumir un solo input/output
			if test.Input != "" && test.ExpectedOutput != "" {
				// Parse input - si es numérico, usar directamente, si no, entre comillas
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
	if input == "" {
		return ""
	}

	// Si es numérico, devolver tal cual
	if g.isNumeric(input) {
		return input
	}

	// Si no, entre comillas como string
	return fmt.Sprintf("\"%s\"", input)
}

// formatExpectedOutput formatea el output esperado para C++
func (g *CppTemplateGenerator) formatExpectedOutput(output string) string {
	output = strings.TrimSpace(output)
	if output == "" {
		return ""
	}

	// Si es numérico, devolver tal cual
	if g.isNumeric(output) {
		return output
	}

	// Si no, entre comillas como string
	return fmt.Sprintf("\"%s\"", output)
}

// isNumeric verifica si una cadena representa un número
func (g *CppTemplateGenerator) isNumeric(s string) bool {
	if s == "" {
		return false
	}

	// Verificar si es un número entero o flotante
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

// buildTemplate construye el template completo reemplazando las secciones
func (g *CppTemplateGenerator) buildTemplate(solutionCode, testCode string) string {
	template := `// Start Test
#define DOCTEST_CONFIG_IMPLEMENT_WITH_MAIN
// Solution - Start
#include "doctest.h"
#include <iostream>
using namespace std;

%s
// Solution - End

// Tests - Start
%s
// Tests - End
`

	return fmt.Sprintf(template, solutionCode, testCode)
}
