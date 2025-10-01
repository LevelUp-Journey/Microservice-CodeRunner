package repository

import (
	"fmt"
	"time"

	"code-runner/internal/database/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type GeneratedTestCodeRepository struct {
	db *gorm.DB
}

// NewGeneratedTestCodeRepository creates a new generated test code repository
func NewGeneratedTestCodeRepository(db *gorm.DB) *GeneratedTestCodeRepository {
	return &GeneratedTestCodeRepository{
		db: db,
	}
}

// Create creates a new generated test code record
func (r *GeneratedTestCodeRepository) Create(generatedTestCode *models.GeneratedTestCode) error {
	if err := r.db.Create(generatedTestCode).Error; err != nil {
		return fmt.Errorf("failed to create generated test code: %w", err)
	}
	return nil
}

// GetByID retrieves a generated test code by its ID
func (r *GeneratedTestCodeRepository) GetByID(id uuid.UUID) (*models.GeneratedTestCode, error) {
	var generatedTestCode models.GeneratedTestCode
	err := r.db.Preload("Execution").First(&generatedTestCode, "id = ?", id).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("generated test code not found with id: %s", id)
		}
		return nil, fmt.Errorf("failed to get generated test code: %w", err)
	}

	return &generatedTestCode, nil
}

// GetByExecutionID retrieves generated test code by execution ID
func (r *GeneratedTestCodeRepository) GetByExecutionID(executionID uuid.UUID) (*models.GeneratedTestCode, error) {
	var generatedTestCode models.GeneratedTestCode
	err := r.db.Where("execution_id = ?", executionID).First(&generatedTestCode).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("generated test code not found for execution: %s", executionID)
		}
		return nil, fmt.Errorf("failed to get generated test code for execution: %w", err)
	}

	return &generatedTestCode, nil
}

// GetByChallengeID retrieves all generated test codes for a challenge
func (r *GeneratedTestCodeRepository) GetByChallengeID(challengeID string, limit, offset int) ([]*models.GeneratedTestCode, error) {
	var generatedTestCodes []*models.GeneratedTestCode
	query := r.db.Where("challenge_id = ?", challengeID).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&generatedTestCodes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get generated test codes for challenge %s: %w", challengeID, err)
	}

	return generatedTestCodes, nil
}

// GetByLanguage retrieves generated test codes by language
func (r *GeneratedTestCodeRepository) GetByLanguage(language string, limit, offset int) ([]*models.GeneratedTestCode, error) {
	var generatedTestCodes []*models.GeneratedTestCode
	query := r.db.Where("language = ?", language).Order("created_at DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	err := query.Find(&generatedTestCodes).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get generated test codes for language %s: %w", language, err)
	}

	return generatedTestCodes, nil
}

// Update updates a generated test code record
func (r *GeneratedTestCodeRepository) Update(generatedTestCode *models.GeneratedTestCode) error {
	if err := r.db.Save(generatedTestCode).Error; err != nil {
		return fmt.Errorf("failed to update generated test code: %w", err)
	}
	return nil
}

// Delete deletes a generated test code record
func (r *GeneratedTestCodeRepository) Delete(id uuid.UUID) error {
	if err := r.db.Delete(&models.GeneratedTestCode{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to delete generated test code: %w", err)
	}
	return nil
}

// CleanupOldRecords deletes generated test codes older than the specified duration
func (r *GeneratedTestCodeRepository) CleanupOldRecords(olderThan time.Duration) error {
	cutoffTime := time.Now().Add(-olderThan)
	result := r.db.Where("created_at < ?", cutoffTime).Delete(&models.GeneratedTestCode{})

	if result.Error != nil {
		return fmt.Errorf("failed to cleanup old generated test codes: %w", result.Error)
	}

	return nil
}
