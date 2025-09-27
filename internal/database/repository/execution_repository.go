package repository

import (
	"fmt"
	"time"

	"code-runner/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ExecutionRepository struct {
	db *gorm.DB
}

// NewExecutionRepository creates a new execution repository
func NewExecutionRepository(db *gorm.DB) *ExecutionRepository {
	return &ExecutionRepository{
		db: db,
	}
}

// Create creates a new execution record
func (r *ExecutionRepository) Create(execution *models.Execution) error {
	if err := r.db.Create(execution).Error; err != nil {
		return fmt.Errorf("failed to create execution: %w", err)
	}
	return nil
}

// GetByID retrieves an execution by its ID
func (r *ExecutionRepository) GetByID(id uuid.UUID) (*models.Execution, error) {
	var execution models.Execution
	err := r.db.Preload("ExecutionSteps").
		Preload("ExecutionLogs").
		Preload("TestResults").
		First(&execution, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("execution not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

// GetBySolutionID retrieves executions by solution ID
func (r *ExecutionRepository) GetBySolutionID(solutionID string, limit int, offset int) ([]models.Execution, error) {
	var executions []models.Execution
	query := r.db.Where("solution_id = ?", solutionID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by solution ID: %w", err)
	}

	return executions, nil
}

// GetByStudentID retrieves executions by student ID
func (r *ExecutionRepository) GetByStudentID(studentID string, limit int, offset int) ([]models.Execution, error) {
	var executions []models.Execution
	query := r.db.Where("student_id = ?", studentID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by student ID: %w", err)
	}

	return executions, nil
}

// GetByChallengeID retrieves executions by challenge ID
func (r *ExecutionRepository) GetByChallengeID(challengeID string, limit int, offset int) ([]models.Execution, error) {
	var executions []models.Execution
	query := r.db.Where("challenge_id = ?", challengeID).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by challenge ID: %w", err)
	}

	return executions, nil
}

// GetByStatus retrieves executions by status
func (r *ExecutionRepository) GetByStatus(status models.ExecutionStatus, limit int, offset int) ([]models.Execution, error) {
	var executions []models.Execution
	query := r.db.Where("status = ?", status).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by status: %w", err)
	}

	return executions, nil
}

// Update updates an execution record
func (r *ExecutionRepository) Update(execution *models.Execution) error {
	if err := r.db.Save(execution).Error; err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}
	return nil
}

// UpdateStatus updates only the status of an execution
func (r *ExecutionRepository) UpdateStatus(id uuid.UUID, status models.ExecutionStatus) error {
	result := r.db.Model(&models.Execution{}).
		Where("id = ?", id).
		Update("status", status)

	if result.Error != nil {
		return fmt.Errorf("failed to update execution status: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("execution not found with id: %s", id)
	}

	return nil
}

// Delete soft deletes an execution
func (r *ExecutionRepository) Delete(id uuid.UUID) error {
	result := r.db.Delete(&models.Execution{}, "id = ?", id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete execution: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("execution not found with id: %s", id)
	}

	return nil
}

// AddExecutionStep adds a step to an execution
func (r *ExecutionRepository) AddExecutionStep(step *models.ExecutionStep) error {
	if err := r.db.Create(step).Error; err != nil {
		return fmt.Errorf("failed to create execution step: %w", err)
	}
	return nil
}

// UpdateExecutionStep updates an execution step
func (r *ExecutionRepository) UpdateExecutionStep(step *models.ExecutionStep) error {
	if err := r.db.Save(step).Error; err != nil {
		return fmt.Errorf("failed to update execution step: %w", err)
	}
	return nil
}

// AddExecutionLog adds a log entry to an execution
func (r *ExecutionRepository) AddExecutionLog(log *models.ExecutionLog) error {
	if err := r.db.Create(log).Error; err != nil {
		return fmt.Errorf("failed to create execution log: %w", err)
	}
	return nil
}

// AddTestResult adds a test result to an execution
func (r *ExecutionRepository) AddTestResult(testResult *models.TestResult) error {
	if err := r.db.Create(testResult).Error; err != nil {
		return fmt.Errorf("failed to create test result: %w", err)
	}
	return nil
}

// GetExecutionStats retrieves execution statistics
func (r *ExecutionRepository) GetExecutionStats(studentID string, challengeID string, dateFrom time.Time, dateTo time.Time) (map[string]interface{}, error) {
	var stats struct {
		Total     int64 `json:"total"`
		Completed int64 `json:"completed"`
		Failed    int64 `json:"failed"`
		Pending   int64 `json:"pending"`
		Running   int64 `json:"running"`
	}

	query := r.db.Model(&models.Execution{})

	// Apply filters
	if studentID != "" {
		query = query.Where("student_id = ?", studentID)
	}
	if challengeID != "" {
		query = query.Where("challenge_id = ?", challengeID)
	}
	if !dateFrom.IsZero() {
		query = query.Where("created_at >= ?", dateFrom)
	}
	if !dateTo.IsZero() {
		query = query.Where("created_at <= ?", dateTo)
	}

	// Get total count
	if err := query.Count(&stats.Total).Error; err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}

	// Get counts by status
	statusQuery := query
	if err := statusQuery.Where("status = ?", models.StatusCompleted).Count(&stats.Completed).Error; err != nil {
		return nil, fmt.Errorf("failed to get completed count: %w", err)
	}

	statusQuery = query
	if err := statusQuery.Where("status = ?", models.StatusFailed).Count(&stats.Failed).Error; err != nil {
		return nil, fmt.Errorf("failed to get failed count: %w", err)
	}

	statusQuery = query
	if err := statusQuery.Where("status = ?", models.StatusPending).Count(&stats.Pending).Error; err != nil {
		return nil, fmt.Errorf("failed to get pending count: %w", err)
	}

	statusQuery = query
	if err := statusQuery.Where("status = ?", models.StatusRunning).Count(&stats.Running).Error; err != nil {
		return nil, fmt.Errorf("failed to get running count: %w", err)
	}

	return map[string]interface{}{
		"total":     stats.Total,
		"completed": stats.Completed,
		"failed":    stats.Failed,
		"pending":   stats.Pending,
		"running":   stats.Running,
	}, nil
}

// GetRecentExecutions retrieves the most recent executions
func (r *ExecutionRepository) GetRecentExecutions(limit int) ([]models.Execution, error) {
	var executions []models.Execution
	err := r.db.Order("created_at DESC").
		Limit(limit).
		Find(&executions).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get recent executions: %w", err)
	}

	return executions, nil
}

// GetExecutionsByDateRange retrieves executions within a date range
func (r *ExecutionRepository) GetExecutionsByDateRange(from time.Time, to time.Time, limit int, offset int) ([]models.Execution, error) {
	var executions []models.Execution
	query := r.db.Where("created_at BETWEEN ? AND ?", from, to).
		Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&executions).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get executions by date range: %w", err)
	}

	return executions, nil
}

// CountByStudentAndChallenge counts executions for a specific student and challenge
func (r *ExecutionRepository) CountByStudentAndChallenge(studentID string, challengeID string) (int64, error) {
	var count int64
	err := r.db.Model(&models.Execution{}).
		Where("student_id = ? AND challenge_id = ?", studentID, challengeID).
		Count(&count).Error

	if err != nil {
		return 0, fmt.Errorf("failed to count executions: %w", err)
	}

	return count, nil
}

// GetExecutionStepByID retrieves an execution step by its ID
func (r *ExecutionRepository) GetExecutionStepByID(stepID uuid.UUID) (*models.ExecutionStep, error) {
	var step models.ExecutionStep
	err := r.db.First(&step, "id = ?", stepID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("execution step not found with id: %s", stepID)
		}
		return nil, fmt.Errorf("failed to get execution step: %w", err)
	}

	return &step, nil
}

// GetExecutionStepsByExecutionID retrieves all steps for an execution
func (r *ExecutionRepository) GetExecutionStepsByExecutionID(executionID uuid.UUID) ([]models.ExecutionStep, error) {
	var steps []models.ExecutionStep
	err := r.db.Where("execution_id = ?", executionID).
		Order("step_order ASC").
		Find(&steps).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get execution steps: %w", err)
	}

	return steps, nil
}

// GetExecutionLogsByExecutionID retrieves all logs for an execution
func (r *ExecutionRepository) GetExecutionLogsByExecutionID(executionID uuid.UUID) ([]models.ExecutionLog, error) {
	var logs []models.ExecutionLog
	err := r.db.Where("execution_id = ?", executionID).
		Order("timestamp ASC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get execution logs: %w", err)
	}

	return logs, nil
}

// GetTestResultsByExecutionID retrieves all test results for an execution
func (r *ExecutionRepository) GetTestResultsByExecutionID(executionID uuid.UUID) ([]models.TestResult, error) {
	var testResults []models.TestResult
	err := r.db.Where("execution_id = ?", executionID).
		Find(&testResults).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get test results: %w", err)
	}

	return testResults, nil
}

// DeleteExecutionStep deletes an execution step
func (r *ExecutionRepository) DeleteExecutionStep(stepID uuid.UUID) error {
	result := r.db.Delete(&models.ExecutionStep{}, "id = ?", stepID)
	if result.Error != nil {
		return fmt.Errorf("failed to delete execution step: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("execution step not found with id: %s", stepID)
	}

	return nil
}
