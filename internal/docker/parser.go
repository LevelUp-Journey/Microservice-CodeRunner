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
}

// DoctestParser implementa TestResultParser para doctest (C++)
type DoctestParser struct{}

// NewDoctestParser crea una nueva instancia del parser para doctest
func NewDoctestParser() *DoctestParser {
	return &DoctestParser{}
}

// Parse implementa el parsing robusto de output de doctest
func (p *DoctestParser) Parse(output string, testIDs []string) ([]TestResult, error) {
	lines := strings.Split(output, "\n")

	// Maps para tracking
	failedTests := make(map[string]bool)
	testResults := make([]TestResult, 0, len(testIDs))

	// Regex patterns para doctest output
	testCasePattern := regexp.MustCompile(`\[doctest\]\s+TEST CASE:\s+(.+)`)
	summaryPattern := regexp.MustCompile(`\[doctest\]\s+test cases:\s+(\d+)\s*\|\s*(\d+)\s+passed\s*\|\s*(\d+)\s+failed`)

	var totalTests, passedTests, failedTestsCount int

	// Parse line by line
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect test case failures
		if matches := testCasePattern.FindStringSubmatch(line); len(matches) > 1 {
			testName := strings.TrimSpace(matches[1])
			// Remove quotes if present
			testName = strings.Trim(testName, `"`)
			failedTests[testName] = true
		}

		// Parse final summary
		if matches := summaryPattern.FindStringSubmatch(line); len(matches) == 4 {
			if t, err := strconv.Atoi(matches[1]); err == nil {
				totalTests = t
			}
			if p, err := strconv.Atoi(matches[2]); err == nil {
				passedTests = p
			}
			if f, err := strconv.Atoi(matches[3]); err == nil {
				failedTestsCount = f
			}
		}
	}

	// Build results for each expected test ID
	for _, testID := range testIDs {
		passed := !failedTests[testID]
		result := TestResult{
			TestID:   testID,
			TestName: testID,
			Passed:   passed,
		}

		// If failed, try to extract error message from nearby lines
		if !passed {
			result.ErrorMessage = p.extractErrorMessage(lines, testID)
		}

		testResults = append(testResults, result)
	}

	// Validation: check if parsed numbers match expectations
	actualPassed := 0
	actualFailed := 0
	for _, result := range testResults {
		if result.Passed {
			actualPassed++
		} else {
			actualFailed++
		}
	}

	if totalTests != len(testIDs) {
		return testResults, fmt.Errorf("test count mismatch: doctest reported %d tests, expected %d", totalTests, len(testIDs))
	}

	if passedTests != actualPassed || failedTestsCount != actualFailed {
		return testResults, fmt.Errorf("result mismatch: doctest reported %d passed/%d failed, but parsing found %d passed/%d failed",
			passedTests, failedTestsCount, actualPassed, actualFailed)
	}

	return testResults, nil
}

// extractErrorMessage intenta extraer el mensaje de error para un test fallido
func (p *DoctestParser) extractErrorMessage(lines []string, testID string) string {
	testCasePattern := regexp.MustCompile(`\[doctest\]\s+TEST CASE:\s+` + regexp.QuoteMeta(testID))
	checkPattern := regexp.MustCompile(`CHECK\(.+\)\s+is NOT correct!`)

	inTestCase := false
	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Start of our test case
		if testCasePattern.MatchString(line) {
			inTestCase = true
			continue
		}

		// End of test case (next test case or summary)
		if inTestCase && (strings.Contains(line, "[doctest] TEST CASE:") ||
			strings.Contains(line, "[doctest] test cases:")) {
			break
		}

		// Extract CHECK failure message
		if inTestCase && checkPattern.MatchString(line) {
			return strings.TrimSpace(line)
		}
	}

	return "Test failed - check output for details"
}
