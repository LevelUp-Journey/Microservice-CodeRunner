package docker

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TestResultParser define la interfaz Strategy para parsear resultados de tests
// dependiendo del lenguaje/framework de testing usado
type TestResultParser interface {
	// Parse analiza el output de ejecución y retorna resultados detallados por test
	Parse(output string, testIDs []string) ([]TestResult, error)
	// CanParse indica si la estrategia puede manejar el lenguaje solicitado
	CanParse(language string) bool
	// DetectsOutput indica si la estrategia reconoce el formato del output de pruebas
	DetectsOutput(output string) bool
}

// DoctestParser implementa TestResultParser para doctest (C++)
type DoctestParser struct{}

// NewDoctestParser crea una nueva instancia del parser para doctest
func NewDoctestParser() *DoctestParser {
	return &DoctestParser{}
}

// CanParse indica si este parser soporta el lenguaje dado
func (p *DoctestParser) CanParse(language string) bool {
	return strings.EqualFold(language, "cpp") || strings.EqualFold(language, "c++")
}

func (p *DoctestParser) DetectsOutput(output string) bool {
	return strings.Contains(output, "[doctest] test cases:")
}

// Parse implementa el parsing robusto de output de doctest
func (p *DoctestParser) Parse(output string, testIDs []string) ([]TestResult, error) {
	lines := strings.Split(output, "\n")

	// Maps para tracking
	failedTests := make(map[string]bool)
	failureMessages := make(map[string][]string)
	testResults := make([]TestResult, 0, len(testIDs))

	// Regex patterns para doctest output
	testCasePattern := regexp.MustCompile(`(?i)(?:\[doctest\]\s+)?TEST CASE:\s+(.+)`)
	summaryPattern := regexp.MustCompile(`\[doctest\]\s+test cases:\s+(\d+)\s*\|\s*(\d+)\s+passed\s*\|\s*(\d+)\s+failed`)
	failurePattern := regexp.MustCompile(`(?i)(?:ERROR:\s+)?CHECK\(.+\)\s+is NOT correct!`)

	var totalTests, passedTests, failedTestsCount int
	var currentTest string
	var summaryDetected bool
	var captureActive bool
	var captureKey string

	// Parse line by line
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if line == "" {
			captureActive = false
			continue
		}

		if matches := testCasePattern.FindStringSubmatch(line); len(matches) > 1 {
			testName := strings.TrimSpace(matches[1])
			testName = strings.Trim(testName, `"`)
			currentTest = testName
			captureActive = false
			continue
		}

		if captureActive {
			if strings.HasPrefix(line, "[doctest]") ||
				strings.HasPrefix(line, "TEST CASE:") ||
				strings.HasPrefix(line, "===============================================================================") {
				captureActive = false
			} else if captureKey != "" {
				failureMessages[captureKey] = append(failureMessages[captureKey], line)
				continue
			}
		}

		if failurePattern.MatchString(line) {
			if currentTest != "" {
				captureKey = normalizeTestIdentifier(currentTest)
				failedTests[captureKey] = true
				failureMessages[captureKey] = append(failureMessages[captureKey], line)
				captureActive = true
			}
			continue
		}

		if matches := summaryPattern.FindStringSubmatch(line); len(matches) == 4 {
			if t, err := strconv.Atoi(matches[1]); err == nil {
				totalTests = t
			}
			if pVal, err := strconv.Atoi(matches[2]); err == nil {
				passedTests = pVal
			}
			if f, err := strconv.Atoi(matches[3]); err == nil {
				failedTestsCount = f
			}
			currentTest = ""
			captureActive = false
			summaryDetected = true
		}
	}

	// Build results for each expected test ID
	for _, testID := range testIDs {
		normID := normalizeTestIdentifier(testID)
		passed := !failedTests[normID]
		result := TestResult{
			TestID:   testID,
			TestName: testID,
			Passed:   passed,
		}

		if !passed {
			if messages := failureMessages[normID]; len(messages) > 0 {
				result.ErrorMessage = strings.Join(messages, "\n")
			} else {
				result.ErrorMessage = p.extractErrorMessage(lines, testID)
			}
		}

		testResults = append(testResults, result)
	}

	// Validation y normalización de conteos
	actualPassed := 0
	actualFailed := 0
	for _, result := range testResults {
		if result.Passed {
			actualPassed++
		} else {
			actualFailed++
		}
	}

	if !summaryDetected {
		totalTests = len(testIDs)
		passedTests = actualPassed
		failedTestsCount = actualFailed
	} else {
		if totalTests != len(testIDs) {
			return testResults, fmt.Errorf("test count mismatch: doctest reported %d tests, expected %d", totalTests, len(testIDs))
		}
		if passedTests != actualPassed || failedTestsCount != actualFailed {
			return testResults, fmt.Errorf(
				"result mismatch: doctest reported %d passed/%d failed, but parsing found %d passed/%d failed",
				passedTests, failedTestsCount, actualPassed, actualFailed,
			)
		}
	}

	return testResults, nil
}

func normalizeTestIdentifier(id string) string {
	return strings.ToLower(strings.TrimSpace(id))
}

// extractErrorMessage intenta extraer el mensaje de error para un test fallido
func (p *DoctestParser) extractErrorMessage(lines []string, testID string) string {
	testCasePattern := regexp.MustCompile(`(?i)(?:\[doctest\]\s+)?TEST CASE:\s+` + regexp.QuoteMeta(testID))
	checkPattern := regexp.MustCompile(`CHECK\(.+\)\s+is NOT correct!`)

	inTestCase := false
	collecting := false
	var messages []string

	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)

		if testCasePattern.MatchString(line) {
			inTestCase = true
			collecting = false
			messages = messages[:0]
			continue
		}

		if !inTestCase {
			continue
		}

		if strings.Contains(line, "[doctest] TEST CASE:") ||
			strings.HasPrefix(line, "TEST CASE:") ||
			strings.Contains(line, "[doctest] test cases:") {
			break
		}

		if collecting {
			if line == "" || strings.HasPrefix(line, "[doctest]") || strings.HasPrefix(line, "===============================================================================") {
				break
			}
			messages = append(messages, line)
			continue
		}

		if checkPattern.MatchString(line) {
			messages = append(messages, line)
			collecting = true
		}
	}

	if len(messages) > 0 {
		return strings.Join(messages, "\n")
	}

	return "Test failed - check output for details"
}
