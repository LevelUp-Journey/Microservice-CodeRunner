package services

import (
	"fmt"
	"log"
	"os"
	"time"

	"code-runner/internal/database/models"
	"code-runner/internal/database/repository"

	"github.com/google/uuid"
)

type ExecutionService struct {
	repo *repository.ExecutionRepository
}

// NewExecutionService creates a new execution service
func NewExecutionService(repo *repository.ExecutionRepository) *ExecutionService {
	return &ExecutionService{
		repo: repo,
	}
}

// CreateExecution creates a new execution record
func (s *ExecutionService) CreateExecution(solutionID, challengeID, studentID, code, language string) (*models.Execution, error) {
	execution := &models.Execution{
		SolutionID:  solutionID,
		ChallengeID: challengeID,
		StudentID:   studentID,
		Language:    language,
		Code:        code,
		Status:      models.StatusPending,
		TotalTests:  0,
		PassedTests: 0,
	}

	// Get server instance name
	if hostname, err := os.Hostname(); err == nil {
		execution.ServerInstance = hostname
	}

	if err := s.repo.Create(execution); err != nil {
		return nil, fmt.Errorf("failed to create execution: %w", err)
	}

	log.Printf("✅ Created execution record: %s for student: %s, challenge: %s",
		execution.ID, studentID, challengeID)

	return execution, nil
}

// StartExecution marks an execution as running and creates the initial step
func (s *ExecutionService) StartExecution(executionID uuid.UUID) error {
	// Update execution status to running
	if err := s.repo.UpdateStatus(executionID, models.StatusRunning); err != nil {
		return fmt.Errorf("failed to update execution status: %w", err)
	}

	// Create initial execution step
	step := &models.ExecutionStep{
		ExecutionID: executionID,
		StepName:    "initialization",
		StepOrder:   0,
		Status:      models.StatusRunning,
		StartedAt:   timePtr(time.Now()),
	}

	if err := s.repo.AddExecutionStep(step); err != nil {
		return fmt.Errorf("failed to create initial step: %w", err)
	}

	// Add log entry
	logEntry := &models.ExecutionLog{
		ExecutionID: executionID,
		Level:       "info",
		Message:     "Execution started",
		Source:      "system",
		Timestamp:   time.Now(),
	}

	if err := s.repo.AddExecutionLog(logEntry); err != nil {
		log.Printf("⚠️ Failed to add log entry: %v", err)
	}

	return nil
}

// AddExecutionStep adds a new step to the execution pipeline
func (s *ExecutionService) AddExecutionStep(executionID uuid.UUID, stepName string, stepOrder int) (*models.ExecutionStep, error) {
	step := &models.ExecutionStep{
		ExecutionID: executionID,
		StepName:    stepName,
		StepOrder:   stepOrder,
		Status:      models.StatusPending,
	}

	if err := s.repo.AddExecutionStep(step); err != nil {
		return nil, fmt.Errorf("failed to add execution step: %w", err)
	}

	return step, nil
}

// StartExecutionStep marks a step as running
func (s *ExecutionService) StartExecutionStep(stepID uuid.UUID) error {
	step, err := s.getExecutionStepByID(stepID)
	if err != nil {
		return err
	}

	now := time.Now()
	step.Status = models.StatusRunning
	step.StartedAt = &now

	if err := s.repo.UpdateExecutionStep(step); err != nil {
		return fmt.Errorf("failed to update execution step: %w", err)
	}

	return nil
}

// CompleteExecutionStep marks a step as completed
func (s *ExecutionService) CompleteExecutionStep(stepID uuid.UUID, metadata string) error {
	step, err := s.getExecutionStepByID(stepID)
	if err != nil {
		return err
	}

	now := time.Now()
	step.Status = models.StatusCompleted
	step.CompletedAt = &now
	step.Metadata = metadata

	// Calculate duration if started
	if step.StartedAt != nil {
		duration := now.Sub(*step.StartedAt).Milliseconds()
		step.DurationMs = &duration
	}

	if err := s.repo.UpdateExecutionStep(step); err != nil {
		return fmt.Errorf("failed to update execution step: %w", err)
	}

	return nil
}

// FailExecutionStep marks a step as failed
func (s *ExecutionService) FailExecutionStep(stepID uuid.UUID, errorMessage string) error {
	step, err := s.getExecutionStepByID(stepID)
	if err != nil {
		return err
	}

	now := time.Now()
	step.Status = models.StatusFailed
	step.CompletedAt = &now
	step.ErrorMessage = errorMessage

	// Calculate duration if started
	if step.StartedAt != nil {
		duration := now.Sub(*step.StartedAt).Milliseconds()
		step.DurationMs = &duration
	}

	if err := s.repo.UpdateExecutionStep(step); err != nil {
		return fmt.Errorf("failed to update execution step: %w", err)
	}

	return nil
}

// CompleteExecution marks an execution as completed with results
func (s *ExecutionService) CompleteExecution(executionID uuid.UUID, success bool, message string,
	approvedTestIDs []string, failedTestIDs []string, executionTimeMs int64, memoryUsageMB float64) error {

	execution, err := s.repo.GetByID(executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	execution.Status = models.StatusCompleted
	execution.Success = success
	execution.Message = message
	execution.ApprovedTestIDs = approvedTestIDs
	execution.FailedTestIDs = failedTestIDs
	execution.TotalTests = len(approvedTestIDs) + len(failedTestIDs)
	execution.PassedTests = len(approvedTestIDs)
	execution.ExecutionTimeMs = &executionTimeMs
	execution.MemoryUsageMB = &memoryUsageMB

	if err := s.repo.Update(execution); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Add completion log
	logLevel := "info"
	if !success {
		logLevel = "warn"
	}

	logEntry := &models.ExecutionLog{
		ExecutionID: executionID,
		Level:       logLevel,
		Message:     fmt.Sprintf("Execution completed. Success: %t, Passed: %d/%d tests", success, len(approvedTestIDs), execution.TotalTests),
		Source:      "system",
		Timestamp:   time.Now(),
	}

	if err := s.repo.AddExecutionLog(logEntry); err != nil {
		log.Printf("⚠️ Failed to add completion log: %v", err)
	}

	log.Printf("✅ Execution %s completed. Success: %t, Tests: %d/%d passed",
		executionID, success, len(approvedTestIDs), execution.TotalTests)

	return nil
}

// FailExecution marks an execution as failed
func (s *ExecutionService) FailExecution(executionID uuid.UUID, errorMessage, errorType string) error {
	execution, err := s.repo.GetByID(executionID)
	if err != nil {
		return fmt.Errorf("failed to get execution: %w", err)
	}

	execution.Status = models.StatusFailed
	execution.Success = false
	execution.ErrorMessage = errorMessage
	execution.ErrorType = errorType

	if err := s.repo.Update(execution); err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}

	// Add error log
	logEntry := &models.ExecutionLog{
		ExecutionID: executionID,
		Level:       "error",
		Message:     fmt.Sprintf("Execution failed: %s", errorMessage),
		Source:      "system",
		Timestamp:   time.Now(),
	}

	if err := s.repo.AddExecutionLog(logEntry); err != nil {
		log.Printf("⚠️ Failed to add error log: %v", err)
	}

	log.Printf("❌ Execution %s failed: %s", executionID, errorMessage)

	return nil
}

// AddTestResult adds a test result to an execution
func (s *ExecutionService) AddTestResult(executionID uuid.UUID, testID, testName, input, expectedOutput,
	actualOutput string, passed bool, executionTimeMs int64, errorMessage string) error {

	testResult := &models.TestResult{
		ExecutionID:     executionID,
		TestID:          testID,
		TestName:        testName,
		Input:           input,
		ExpectedOutput:  expectedOutput,
		ActualOutput:    actualOutput,
		Passed:          passed,
		ExecutionTimeMs: &executionTimeMs,
		ErrorMessage:    errorMessage,
	}

	if err := s.repo.AddTestResult(testResult); err != nil {
		return fmt.Errorf("failed to add test result: %w", err)
	}

	return nil
}

// AddLog adds a log entry to an execution
func (s *ExecutionService) AddLog(executionID uuid.UUID, level, message, source string) error {
	logEntry := &models.ExecutionLog{
		ExecutionID: executionID,
		Level:       level,
		Message:     message,
		Source:      source,
		Timestamp:   time.Now(),
	}

	if err := s.repo.AddExecutionLog(logEntry); err != nil {
		return fmt.Errorf("failed to add log entry: %w", err)
	}

	return nil
}

// GetExecution retrieves an execution by ID
func (s *ExecutionService) GetExecution(executionID uuid.UUID) (*models.Execution, error) {
	return s.repo.GetByID(executionID)
}

// GetExecutionsByStudent retrieves executions for a student
func (s *ExecutionService) GetExecutionsByStudent(studentID string, limit, offset int) ([]models.Execution, error) {
	return s.repo.GetByStudentID(studentID, limit, offset)
}

// GetExecutionsByChallenge retrieves executions for a challenge
func (s *ExecutionService) GetExecutionsByChallenge(challengeID string, limit, offset int) ([]models.Execution, error) {
	return s.repo.GetByChallengeID(challengeID, limit, offset)
}

// GetExecutionStats retrieves execution statistics
func (s *ExecutionService) GetExecutionStats(studentID, challengeID string, dateFrom, dateTo time.Time) (map[string]interface{}, error) {
	return s.repo.GetExecutionStats(studentID, challengeID, dateFrom, dateTo)
}

// Helper method to get execution step by ID
func (s *ExecutionService) getExecutionStepByID(stepID uuid.UUID) (*models.ExecutionStep, error) {
	return s.repo.GetExecutionStepByID(stepID)
}

// GetExecutionSteps retrieves all steps for an execution
func (s *ExecutionService) GetExecutionSteps(executionID uuid.UUID) ([]models.ExecutionStep, error) {
	return s.repo.GetExecutionStepsByExecutionID(executionID)
}

// GetExecutionLogs retrieves all logs for an execution
func (s *ExecutionService) GetExecutionLogs(executionID uuid.UUID) ([]models.ExecutionLog, error) {
	return s.repo.GetExecutionLogsByExecutionID(executionID)
}

// GetTestResults retrieves all test results for an execution
func (s *ExecutionService) GetTestResults(executionID uuid.UUID) ([]models.TestResult, error) {
	return s.repo.GetTestResultsByExecutionID(executionID)
}

// DeleteExecutionStep removes an execution step
func (s *ExecutionService) DeleteExecutionStep(stepID uuid.UUID) error {
	return s.repo.DeleteExecutionStep(stepID)
}

// Helper function to create time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
