package docker

import (
	"fmt"
	"log"
	"strings"
)

// parseTestResults parsea la salida de doctest para extraer resultados de tests individuales
// Estrategia:
// 1. Doctest solo muestra en el output los tests que FALLAN
// 2. Los que pasan no aparecen en el output
// 3. Extraer los UUIDs de los tests que aparecen en el output (fallidos)
// 4. Los tests que NO aparecen = PASARON
func (e *DockerExecutor) parseTestResults(result *ExecutionResult, testIDs []string) {
	output := result.StdOut
	lines := strings.Split(output, "\n")

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("🔍 PARSING TEST RESULTS")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📋 Expected tests: %d", len(testIDs))
	log.Printf("📝 Test IDs to validate:")
	for i, id := range testIDs {
		log.Printf("   %d. %s", i+1, id)
	}
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	failedTestIDs := make(map[string]bool)
	var totalTestCases, passedTestCases, failedTestCases int

	log.Printf("\n🔎 ANALYZING DOCTEST OUTPUT:")
	log.Printf("─────────────────────────────────────────────────────")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Detect failed tests
		if strings.Contains(line, "TEST CASE:") {
			parts := strings.Split(line, "TEST CASE:")
			if len(parts) == 2 {
				testID := strings.TrimSpace(parts[1])
				if len(testID) >= 36 && strings.Count(testID, "-") >= 4 {
					failedTestIDs[testID] = true
					log.Printf("   ❌ FAILED TEST DETECTED: %s", testID)
				}
			}
		}

		// Parse doctest summary
		if strings.Contains(line, "test cases:") {
			log.Printf("\n📊 DOCTEST SUMMARY LINE: %s", line)
			parts := strings.Split(line, "|")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if strings.Contains(part, "passed") {
					fmt.Sscanf(part, "%d passed", &passedTestCases)
				} else if strings.Contains(part, "failed") {
					fmt.Sscanf(part, "%d failed", &failedTestCases)
				}
			}
			if len(parts) > 0 {
				firstPart := strings.TrimSpace(parts[0])
				if idx := strings.LastIndex(firstPart, ":"); idx != -1 {
					fmt.Sscanf(firstPart[idx+1:], "%d", &totalTestCases)
				}
			}
		}
	}

	log.Printf("─────────────────────────────────────────────────────")

	result.TotalTests = totalTestCases
	result.PassedTests = passedTestCases
	result.FailedTests = failedTestCases

	log.Printf("\n📈 EXECUTION STATISTICS:")
	log.Printf("   Total test cases: %d", totalTestCases)
	log.Printf("   ✅ Passed: %d", passedTestCases)
	log.Printf("   ❌ Failed: %d", failedTestCases)
	log.Printf("   🔍 Failed test IDs found in output: %d", len(failedTestIDs))

	log.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ MATCHING TEST IDs WITH RESULTS:")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	passedIDs := []string{}
	failedIDs := []string{}

	for i, testID := range testIDs {
		passed := !failedTestIDs[testID]
		result.TestResults = append(result.TestResults, TestResult{
			TestID:   testID,
			TestName: testID,
			Passed:   passed,
		})

		if passed {
			passedIDs = append(passedIDs, testID)
			log.Printf("   ✅ Test #%d: PASSED", i+1)
			log.Printf("      ID: %s", testID)
		} else {
			failedIDs = append(failedIDs, testID)
			log.Printf("   ❌ Test #%d: FAILED", i+1)
			log.Printf("      ID: %s", testID)
		}
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	passedCount := len(passedIDs)
	failedCount := len(failedIDs)

	log.Printf("\n🎯 FINAL RESULTS SUMMARY:")
	log.Printf("   Total: %d tests", len(testIDs))
	log.Printf("   ✅ Passed: %d tests (%d%%)", passedCount, (passedCount*100)/max(len(testIDs), 1))
	log.Printf("   ❌ Failed: %d tests (%d%%)", failedCount, (failedCount*100)/max(len(testIDs), 1))

	if passedCount > 0 {
		log.Printf("\n✅ PASSED TEST IDs:")
		for i, id := range passedIDs {
			log.Printf("   %d. %s", i+1, id)
		}
	}

	if failedCount > 0 {
		log.Printf("\n❌ FAILED TEST IDs:")
		for i, id := range failedIDs {
			log.Printf("   %d. %s", i+1, id)
		}
	}

	if passedCount != passedTestCases {
		log.Printf("\n⚠️  WARNING: Mismatch detected!")
		log.Printf("   Expected passed (from doctest): %d", passedTestCases)
		log.Printf("   Actual passed (from parsing): %d", passedCount)
		log.Printf("   Difference: %d", passedTestCases-passedCount)
	} else {
		log.Printf("\n✅ Consistency check: PASSED")
		log.Printf("   Parsed results match doctest summary")
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
}

// max retorna el máximo entre dos enteros
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
