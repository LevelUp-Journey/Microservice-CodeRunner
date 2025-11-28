package server

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"

	pb "code-runner/api/gen/proto"
	"code-runner/internal/types"
)

// extractFunctionName extrae el nombre de la función del código C++
func extractFunctionName(code string) string {
	// Regex para detectar declaraciones de funciones tipo "int functionName(int param)"
	re := regexp.MustCompile(`(?m)^\s*(?:int|void|double|float|char|string|bool)\s+(\w+)\s*\(`)
	matches := re.FindStringSubmatch(code)
	if len(matches) >= 2 {
		return matches[1]
	}
	return "unknownFunction"
}

// generateCppFile genera un archivo C++ con tests basándose en una plantilla
func generateCppFile(code string, tests []*pb.TestCase) (string, error) {
	// Leer plantilla
	template, err := os.ReadFile("./template/cpp-template.cpp")
	if err != nil {
		return "", fmt.Errorf("failed to read template: %v", err)
	}

	templateStr := string(template)

	// Extraer nombre de función
	functionName := extractFunctionName(code)
	log.Printf("  📝 Extracted function name: %s", functionName)

	// Reemplazar sección de solución
	solutionStart := "// Solution - Start"
	solutionEnd := "// Solution - End"
	solutionPlaceholder := solutionStart + "\n" + code + "\n" + solutionEnd

	// Buscar y reemplazar la sección de solución
	solutionRegex := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(solutionStart) + `.*?` + regexp.QuoteMeta(solutionEnd))
	templateStr = solutionRegex.ReplaceAllString(templateStr, solutionPlaceholder)

	// Generar contenido de tests
	var testsContent strings.Builder
	for i, test := range tests {
		if i > 0 {
			testsContent.WriteString("\n")
		}

		input := test.Input
		expected := test.ExpectedOutput

		// Intentar parsear input como entero
		inputInt, err := strconv.Atoi(input)
		if err != nil {
			// Si no es un entero, tratarlo como string
			input = fmt.Sprintf("\"%s\"", input)
		} else {
			input = strconv.Itoa(inputInt)
		}

		// Intentar parsear expected como entero
		expectedInt, err := strconv.Atoi(expected)
		if err != nil {
			// Si no es un entero, tratarlo como string
			expected = fmt.Sprintf("\"%s\"", expected)
		} else {
			expected = strconv.Itoa(expectedInt)
		}

		testsContent.WriteString(fmt.Sprintf("    CHECK(%s(%s) == %s);", functionName, input, expected))
	}

	// Reemplazar sección de tests
	testsStart := "// Tests - Start"
	testsEnd := "// Tests - End"
	testsPlaceholder := testsStart + "\n" + "TEST_CASE(\"Test_id\") {\n" + testsContent.String() + "\n" + "}\n" + testsEnd
	testsRegex := regexp.MustCompile(`(?s)` + regexp.QuoteMeta(testsStart) + `.*?` + regexp.QuoteMeta(testsEnd))
	templateStr = testsRegex.ReplaceAllString(templateStr, testsPlaceholder)

	// Crear archivo temporal
	tempFile := os.TempDir() + "/generated_code.cpp"
	err = os.WriteFile(tempFile, []byte(templateStr), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return tempFile, nil
}

// convertTestCases convierte test cases de proto a tipos internos
func convertTestCases(protoTests []*pb.TestCase) []*types.TestCase {
	tests := make([]*types.TestCase, len(protoTests))
	for i, pt := range protoTests {
		// Helper function to parse UUID with fallback
		parseTestUUID := func(id string) uuid.UUID {
			if id == "" {
				log.Printf("⚠️  Test ID is empty, generating new UUID")
				return uuid.New()
			}

			if parsed, err := uuid.Parse(id); err == nil {
				return parsed
			}

			log.Printf("⚠️  Invalid test ID format: %s, generating new UUID", id)
			return uuid.New()
		}

		testID := parseTestUUID(pt.CodeVersionTestId)
		codeVersionTestID := parseTestUUID(pt.CodeVersionTestId)

		tests[i] = &types.TestCase{
			TestID:               testID,
			CodeVersionTestID:    codeVersionTestID,
			Input:                pt.Input,
			ExpectedOutput:       pt.ExpectedOutput,
			CustomValidationCode: pt.CustomValidationCode,
		}
	}
	return tests
}
