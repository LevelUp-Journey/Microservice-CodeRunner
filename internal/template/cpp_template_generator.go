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
		ChallengeID:         req.CodeVersionID.String(), // Convertir UUID a string
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

	for i, test := range tests {
		testCount++
		testID := test.CodeVersionTestID
		if testID == uuid.Nil {
			testID = uuid.New()
		}

		if test.HasCustomValidation() {
			// Usar validación personalizada
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID.String()))
			testLines = append(testLines, test.CustomValidationCode)
			testLines = append(testLines, "}")
		} else {
			// Usar input/output estándar
			testLines = append(testLines, fmt.Sprintf(`TEST_CASE("%s") {`, testID.String()))
			// Para casos simples, asumir un solo input/output
			if test.Input != "" && test.ExpectedOutput != "" {
				// Parse input - generar código de setup si hay arrays
				setupCode, functionCall := g.parseComplexInput(test.Input, functionName, i)
				expected := g.formatExpectedOutput(test.ExpectedOutput)

				if setupCode != "" {
					testLines = append(testLines, setupCode)
				}
				testLines = append(testLines, fmt.Sprintf("    CHECK(%s == %s);", functionCall, expected))
			}
			testLines = append(testLines, "}")
		}
	}

	return strings.Join(testLines, "\n"), testCount
}

// parseComplexInput parsea inputs complejos que pueden incluir arrays y múltiples parámetros
func (g *CppTemplateGenerator) parseComplexInput(input string, functionName string, testIndex int) (setupCode string, functionCall string) {
	input = strings.TrimSpace(input)

	// Detectar si hay arrays en el input (formato: [1,2,3] o similar)
	arrayRegex := regexp.MustCompile(`\[([^\]]+)\]`)

	if arrayRegex.MatchString(input) {
		// Hay un array, necesitamos generar código de setup
		var setupLines []string
		var args []string

		// Split por comas fuera de corchetes
		parts := g.splitInputParameters(input)

		for _, part := range parts {
			part = strings.TrimSpace(part)

			if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
				// Es un array
				arrayContent := part[1 : len(part)-1] // Remover [ y ]
				arrayVarName := fmt.Sprintf("arr%d", testIndex)

				// Generar declaración del array
				setupLines = append(setupLines, fmt.Sprintf("    int %s[] = {%s};", arrayVarName, arrayContent))

				// Calcular tamaño del array
				elements := strings.Split(arrayContent, ",")
				arraySize := len(elements)

				// Agregar array como argumento
				args = append(args, arrayVarName)
				// Agregar tamaño como siguiente argumento
				args = append(args, fmt.Sprintf("%d", arraySize))
			} else {
				// Es un parámetro simple
				args = append(args, g.formatSimpleParameter(part))
			}
		}

		setupCode = strings.Join(setupLines, "\n")
		functionCall = fmt.Sprintf("%s(%s)", functionName, strings.Join(args, ", "))
		return setupCode, functionCall
	}

	// No hay arrays, usar formatInput original pero soportar múltiples parámetros
	parts := strings.Split(input, ",")
	if len(parts) > 1 {
		var formattedParts []string
		for _, part := range parts {
			formattedParts = append(formattedParts, g.formatSimpleParameter(strings.TrimSpace(part)))
		}
		functionCall = fmt.Sprintf("%s(%s)", functionName, strings.Join(formattedParts, ", "))
		return "", functionCall
	}

	// Input simple, un solo parámetro
	functionCall = fmt.Sprintf("%s(%s)", functionName, g.formatSimpleParameter(input))
	return "", functionCall
}

// splitInputParameters divide el input en parámetros respetando los corchetes de arrays
func (g *CppTemplateGenerator) splitInputParameters(input string) []string {
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
			// Coma fuera de corchetes, es un separador de parámetros
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteRune(char)
		}
	}

	// Agregar el último parámetro
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

// formatSimpleParameter formatea un parámetro simple (no array)
func (g *CppTemplateGenerator) formatSimpleParameter(param string) string {
	param = strings.TrimSpace(param)

	if param == "" {
		return ""
	}

	// Si es numérico, devolver tal cual
	if g.isNumeric(param) {
		return param
	}

	// Si es booleano
	if param == "true" || param == "false" {
		return param
	}

	// Si no, entre comillas como string
	return fmt.Sprintf("\"%s\"", param)
}

// formatInput formatea el input para C++ (mantenida para compatibilidad)
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

	// Si es un valor booleano, convertir a bool de C++
	if output == "true" || output == "false" {
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
#include "doctest.h"

// Solution - Start
%s
// Solution - End

// Tests - Start
%s
// Tests - End
`

	return fmt.Sprintf(template, solutionCode, testCode)
}
