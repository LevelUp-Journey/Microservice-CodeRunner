package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

// Request represents the structure of the request JSON files
type Request struct {
	ChallengeID   string `json:"challengeId"`
	CodeVersionID string `json:"codeVersionId"`
	StudentID     string `json:"studentId"`
	Code          string `json:"code"`
	Tests         []Test `json:"tests"`
}

// Test represents a test case in the request
type Test struct {
	CodeVersionTestID string `json:"codeVersionTestId"`
	Input             string `json:"input"`
	ExpectedOutput    string `json:"expectedOutput"`
}

func main() {
	requestsDir := "test/exercises_cpp/requests"

	// Read all files in the requests directory
	files, err := filepath.Glob(filepath.Join(requestsDir, "*.json"))
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	// Generate consistent UUIDs for common fields
	challengeID := uuid.New().String()
	codeVersionID := uuid.New().String()
	studentID := uuid.New().String()

	fmt.Printf("Using ChallengeID: %s\n", challengeID)
	fmt.Printf("Using CodeVersionID: %s\n", codeVersionID)
	fmt.Printf("Using StudentID: %s\n", studentID)
	fmt.Println()

	for _, file := range files {
		fmt.Printf("Processing %s...\n", filepath.Base(file))

		// Read the file
		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("Error reading file %s: %v\n", file, err)
			continue
		}

		// Parse JSON
		var request Request
		if err := json.Unmarshal(data, &request); err != nil {
			fmt.Printf("Error parsing JSON in %s: %v\n", file, err)
			continue
		}

		// Update IDs with UUIDs
		request.ChallengeID = challengeID
		request.CodeVersionID = codeVersionID
		request.StudentID = studentID

		// Update test IDs
		for i := range request.Tests {
			request.Tests[i].CodeVersionTestID = uuid.New().String()
		}

		// Write back to file
		updatedData, err := json.MarshalIndent(request, "", "  ")
		if err != nil {
			fmt.Printf("Error marshaling JSON for %s: %v\n", file, err)
			continue
		}

		if err := os.WriteFile(file, updatedData, 0644); err != nil {
			fmt.Printf("Error writing file %s: %v\n", file, err)
			continue
		}

		fmt.Printf("Updated %s successfully\n", filepath.Base(file))
	}

	fmt.Println("\nAll request files have been updated with UUIDs!")
}
