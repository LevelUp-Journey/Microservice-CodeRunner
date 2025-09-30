package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// GeneratedTestCode represents generated test code for auditing and debugging
type GeneratedTestCode struct {
	BaseModel

	// Reference to execution
	ExecutionID uuid.UUID  `gorm:"type:uuid;not null;index" json:"execution_id"`
	Execution   *Execution `gorm:"foreignKey:ExecutionID;constraint:OnDelete:CASCADE" json:"execution,omitempty"`

	// Test code information
	Language      string `gorm:"type:varchar(50);not null;index" json:"language"`
	GeneratorType string `gorm:"type:varchar(50);not null;index" json:"generator_type"`
	TestCode      string `gorm:"type:text;not null" json:"test_code"`

	// Metadata
	ChallengeID         string `gorm:"type:varchar(255);index" json:"challenge_id"`
	TestCasesCount      int    `gorm:"type:integer;default:0" json:"test_cases_count"`
	HasCustomValidation bool   `gorm:"type:boolean;default:false" json:"has_custom_validation"`

	// Performance metrics
	GenerationTimeMS int64 `gorm:"type:bigint" json:"generation_time_ms"`
	CodeSizeBytes    int   `gorm:"type:integer" json:"code_size_bytes"`
}

// TableName returns the table name for GORM
func (GeneratedTestCode) TableName() string {
	return "generated_test_code"
}

// BeforeCreate hook to set code size before creating record
func (gtc *GeneratedTestCode) BeforeCreate(tx *gorm.DB) error {
	// Call base model hook first
	if err := gtc.BaseModel.BeforeCreate(tx); err != nil {
		return err
	}

	// Set code size if not already set
	if gtc.CodeSizeBytes == 0 {
		gtc.CodeSizeBytes = len(gtc.TestCode)
	}

	return nil
}

// NewGeneratedTestCode creates a new GeneratedTestCode instance
func NewGeneratedTestCode(executionID uuid.UUID, language, generatorType, testCode string) *GeneratedTestCode {
	return &GeneratedTestCode{
		ExecutionID:   executionID,
		Language:      language,
		GeneratorType: generatorType,
		TestCode:      testCode,
		CodeSizeBytes: len(testCode),
	}
}

// WithMetadata adds metadata to the GeneratedTestCode
func (gtc *GeneratedTestCode) WithMetadata(challengeID string, testCasesCount int, hasCustomValidation bool) *GeneratedTestCode {
	gtc.ChallengeID = challengeID
	gtc.TestCasesCount = testCasesCount
	gtc.HasCustomValidation = hasCustomValidation
	return gtc
}

// WithGenerationTime sets the generation time
func (gtc *GeneratedTestCode) WithGenerationTime(generationTime time.Duration) *GeneratedTestCode {
	gtc.GenerationTimeMS = generationTime.Milliseconds()
	return gtc
}
