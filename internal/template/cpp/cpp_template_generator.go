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

	// Extraer nombre de función y tipo de retorno usando regex
	functionName, returnType, err := g.extractFunctionInfo(req.Code)
	if err != nil {
		return nil, fmt.Errorf("error extracting function name: %w", err)
	}

	// Generar código de tests
	testCode, testCount := g.generateTestCode(req.TestCases, functionName, returnType)

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

// extractFunctionInfo extrae el nombre y tipo de retorno de la función principal
func (g *CppTemplateGenerator) extractFunctionInfo(code string) (string, string, error) {
	// Regex mejorada para capturar tipo de retorno y nombre de función
	// Soporta:
	//   - Tipos básicos: int, char, double, float, etc.
	//   - Tipos stdint: int64_t, uint32_t, size_t, etc.
	//   - Tipos STL: vector<T>, std::string, etc.
	//   - Modificadores: const, unsigned, *, &
	// Grupo 1: tipo completo
	// Grupo 2: nombre de la función
	re := regexp.MustCompile(`(?m)^\s*((?:const\s+)?(?:unsigned\s+)?(?:int|void|double|float|char|string|bool|auto|long|short|size_t|int8_t|int16_t|int32_t|int64_t|uint8_t|uint16_t|uint32_t|uint64_t|vector<[^>]+>|std::string|std::vector<[^>]+>)(?:\s*\*|\s*&)?)\s+(\w+)\s*\([^)]*\)\s*\{`)
	matches := re.FindStringSubmatch(code)

	if len(matches) < 3 {
		return "", "", fmt.Errorf("no se encontró una función válida en el código")
	}

	returnType := strings.TrimSpace(matches[1])
	functionName := matches[2]

	return functionName, returnType, nil
}

// extractFunctionName usa regex para encontrar el nombre de la función principal (backward compatibility)
func (g *CppTemplateGenerator) extractFunctionName(code string) (string, error) {
	name, _, err := g.extractFunctionInfo(code)
	return name, err
}

// generateTestCode genera el código de tests basado en los test cases
func (g *CppTemplateGenerator) generateTestCode(tests []*types.TestCase, functionName string, returnType string) (string, int) {
	var testLines []string
	testCount := 0

	// Detectar si el tipo de retorno es char* o const char* (punteros a string)
	isStringReturn := strings.Contains(returnType, "char") && strings.Contains(returnType, "*")

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

				// Si retorna char*, usar strcmp para comparar strings
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

				// Detectar el tipo de array (números o strings)
				isStringArray := strings.Contains(arrayContent, "\"")

				if isStringArray {
					// Array de strings: ["flower","flow","flight"]
					// Contar elementos respetando las comillas
					arraySize := g.countArrayElements(arrayContent)

					// Generar declaración de array de char*
					setupLines = append(setupLines, fmt.Sprintf("    char* %s[] = {%s};", arrayVarName, arrayContent))

					// Agregar array y tamaño como argumentos
					args = append(args, arrayVarName)
					args = append(args, fmt.Sprintf("%d", arraySize))
				} else {
					// Array de números: [1,2,3]
					elements := strings.Split(arrayContent, ",")
					arraySize := len(elements)

					// Generar declaración de array de int
					setupLines = append(setupLines, fmt.Sprintf("    int %s[] = {%s};", arrayVarName, arrayContent))

					// Agregar array y tamaño como argumentos
					args = append(args, arrayVarName)
					args = append(args, fmt.Sprintf("%d", arraySize))
				}
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

// countArrayElements cuenta elementos en un array respetando comillas
// Ejemplo: "flower","flow","flight" -> 3
func (g *CppTemplateGenerator) countArrayElements(arrayContent string) int {
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
#include <cstring>

// Solution - Start
%s
// Solution - End

// Tests - Start
%s
// Tests - End
`

	return fmt.Sprintf(template, solutionCode, testCode)
}
