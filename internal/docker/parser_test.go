package docker

import (
	"strings"
	"testing"
)

func TestDoctestParser_Parse_Success(t *testing.T) {
	parser := NewDoctestParser()

	// Sample doctest output with all tests passing
	output := `[doctest] doctest version is "2.4.11"
[doctest] run with "--help" for options
===============================================================================
[doctest] test cases: 2 | 2 passed | 0 failed
[doctest] assertions: 2 | 2 passed | 0 failed
[doctest] Status: SUCCESS
===============================================================================`

	testIDs := []string{"test-uuid-1", "test-uuid-2"}

	results, err := parser.Parse(output, testIDs)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	for _, result := range results {
		if !result.Passed {
			t.Errorf("Expected test %s to pass", result.TestID)
		}
		if result.ErrorMessage != "" {
			t.Errorf("Expected no error message for passed test, got: %s", result.ErrorMessage)
		}
	}
}

func TestDoctestParser_Parse_WithFailures(t *testing.T) {
	parser := NewDoctestParser()

	// Sample doctest output with one test failing
	output := `[doctest] doctest version is "2.4.11"
[doctest] run with "--help" for options
===============================================================================
[doctest] TEST CASE: test-uuid-2
/workspace/solution.cpp:15: CHECK( factorial(-1) == -1 ) is NOT correct!
  values: CHECK( 1 == -1 )
===============================================================================
[doctest] test cases: 2 | 1 passed | 1 failed
[doctest] assertions: 2 | 1 passed | 1 failed
[doctest] Status: FAILURE
===============================================================================`

	testIDs := []string{"test-uuid-1", "test-uuid-2"}

	results, err := parser.Parse(output, testIDs)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	// Check first test passed
	if !results[0].Passed || results[0].TestID != "test-uuid-1" {
		t.Errorf("Expected test-uuid-1 to pass")
	}

	// Check second test failed
	if results[1].Passed || results[1].TestID != "test-uuid-2" {
		t.Errorf("Expected test-uuid-2 to fail")
	}

	if !strings.Contains(results[1].ErrorMessage, "is NOT correct!") {
		t.Errorf("Expected error message to contain failure details, got: %s", results[1].ErrorMessage)
	}
}

func TestDoctestParser_Parse_TestCountMismatch(t *testing.T) {
	parser := NewDoctestParser()

	// Doctest reports 3 tests but we expect 2
	output := `[doctest] doctest version is "2.4.11"
[doctest] test cases: 3 | 2 passed | 1 failed
[doctest] assertions: 3 | 2 passed | 1 failed
[doctest] Status: FAILURE
===============================================================================`

	testIDs := []string{"test-uuid-1", "test-uuid-2"}

	_, err := parser.Parse(output, testIDs)
	if err == nil {
		t.Fatal("Expected error due to test count mismatch")
	}

	if !strings.Contains(err.Error(), "test count mismatch") {
		t.Errorf("Expected error about test count mismatch, got: %v", err)
	}
}

func TestDoctestParser_Parse_ResultMismatch(t *testing.T) {
	parser := NewDoctestParser()

	// Doctest reports 1 passed | 1 failed, but parser finds 0 passed | 2 failed
	output := `[doctest] doctest version is "2.4.11"
[doctest] TEST CASE: test-uuid-1
[doctest] TEST CASE: test-uuid-2
[doctest] test cases: 2 | 1 passed | 1 failed
[doctest] assertions: 2 | 1 passed | 1 failed
[doctest] Status: FAILURE
===============================================================================`

	testIDs := []string{"test-uuid-1", "test-uuid-2"}

	_, err := parser.Parse(output, testIDs)
	if err == nil {
		t.Fatal("Expected error due to result mismatch")
	}

	if !strings.Contains(err.Error(), "result mismatch") {
		t.Errorf("Expected error about result mismatch, got: %v", err)
	}
}

func TestDoctestParser_ExtractErrorMessage(t *testing.T) {
	parser := NewDoctestParser()

	lines := []string{
		"[doctest] TEST CASE: test-uuid-1",
		"/workspace/solution.cpp:10: CHECK( add(2, 3) == 6 ) is NOT correct!",
		"  values: CHECK( 5 == 6 )",
		"[doctest] TEST CASE: test-uuid-2",
	}

	message := parser.extractErrorMessage(lines, "test-uuid-1")
	if !strings.Contains(message, "is NOT correct!") {
		t.Errorf("Expected error message extraction, got: %s", message)
	}

	// Test with non-existent test ID
	message = parser.extractErrorMessage(lines, "test-uuid-3")
	if message != "Test failed - check output for details" {
		t.Errorf("Expected default error message, got: %s", message)
	}
}

func TestDoctestParser_Parse_QuotedTestNames(t *testing.T) {
	parser := NewDoctestParser()

	// Test with quoted test names
	output := `[doctest] doctest version is "2.4.11"
[doctest] TEST CASE: "test-uuid-1"
/workspace/solution.cpp:12: CHECK( add(2, 3) == 6 ) is NOT correct!
  values: CHECK( 5 == 6 )
===============================================================================
[doctest] test cases: 1 | 0 passed | 1 failed
[doctest] assertions: 1 | 0 passed | 1 failed
[doctest] Status: FAILURE
===============================================================================`

	testIDs := []string{"test-uuid-1"}

	results, err := parser.Parse(output, testIDs)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != 1 || results[0].Passed {
		t.Errorf("Expected test to fail")
	}
}

func TestDoctestParser_Parse_UnprefixedTestCase(t *testing.T) {
	parser := NewDoctestParser()

	output := `[doctest] doctest version is "2.4.11"
solution.cpp:27:
TEST CASE:  6d801fc5-4563-4cc3-a546-e444dade79f9

solution.cpp:28: ERROR: CHECK( addOne(-2) == -3 ) is NOT correct!
  values: CHECK( 0 == -3 )

===============================================================================
[doctest] test cases: 3 | 2 passed | 1 failed | 0 skipped
[doctest] assertions: 3 | 2 passed | 1 failed |
[doctest] Status: FAILURE!
===============================================================================`

	testIDs := []string{"test-uuid-1", "6d801fc5-4563-4cc3-a546-e444dade79f9", "test-uuid-3"}

	results, err := parser.Parse(output, testIDs)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if len(results) != len(testIDs) {
		t.Fatalf("Expected %d results, got %d", len(testIDs), len(results))
	}

	if !results[0].Passed {
		t.Errorf("Expected test %s to pass", results[0].TestID)
	}

	if results[1].Passed {
		t.Errorf("Expected test %s to fail", results[1].TestID)
	}
	if !strings.Contains(results[1].ErrorMessage, "is NOT correct!") {
		t.Errorf("Expected detailed failure message, got: %s", results[1].ErrorMessage)
	}

	if !results[2].Passed {
		t.Errorf("Expected test %s to pass", results[2].TestID)
	}
}
