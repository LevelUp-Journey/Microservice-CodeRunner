package repository

import (
	"code-runner/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ExecutionRepository handles database operations for executions
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
	return r.db.Create(execution).Error
}

// Update updates an existing execution record
func (r *ExecutionRepository) Update(execution *models.Execution) error {
	return r.db.Save(execution).Error
}

// GetByID retrieves an execution by ID
func (r *ExecutionRepository) GetByID(id uuid.UUID) (*models.Execution, error) {
	var execution models.Execution
	err := r.db.Preload("GeneratedTestCodes").First(&execution, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &execution, nil
}

// GetBySolutionID retrieves executions by solution ID
func (r *ExecutionRepository) GetBySolutionID(solutionID string) ([]*models.Execution, error) {
	var executions []*models.Execution
	err := r.db.Where("solution_id = ?", solutionID).Order("created_at DESC").Find(&executions).Error
	return executions, err
}

// GetByStudentID retrieves executions by student ID
func (r *ExecutionRepository) GetByStudentID(studentID string) ([]*models.Execution, error) {
	var executions []*models.Execution
	err := r.db.Where("student_id = ?", studentID).Order("created_at DESC").Find(&executions).Error
	return executions, err
}

// UpdateStatus updates the status of an execution
func (r *ExecutionRepository) UpdateStatus(id uuid.UUID, status models.ExecutionStatus) error {
	return r.db.Model(&models.Execution{}).Where("id = ?", id).Update("status", status).Error
}

// UpdateResults updates the execution results
func (r *ExecutionRepository) UpdateResults(id uuid.UUID, success bool, approvedTestIDs, failedTestIDs []string, message string) error {
	updates := map[string]interface{}{
		"success":           success,
		"approved_test_ids": approvedTestIDs,
		"failed_test_ids":   failedTestIDs,
		"passed_tests":      len(approvedTestIDs),
		"total_tests":       len(approvedTestIDs) + len(failedTestIDs),
		"message":           message,
		"status":            models.StatusCompleted,
	}
	return r.db.Model(&models.Execution{}).Where("id = ?", id).Updates(updates).Error
}
