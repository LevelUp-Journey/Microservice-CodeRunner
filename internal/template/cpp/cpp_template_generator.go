package template

import (
	"fmt"
	"time"

	"code-runner/internal/database/models"
	"code-runner/internal/database/repository"
	"code-runner/internal/types"

	"github.com/google/uuid"
)

// CppTemplateGenerator generates C++ templates based on ExecutionRequest
type CppTemplateGenerator struct {
	repo            *repository.GeneratedTestCodeRepository
	functionParser  *FunctionParser
	testGenerator   *TestGenerator
	templateBuilder *TemplateBuilder
}

// NewCppTemplateGenerator creates a new instance of the generator
func NewCppTemplateGenerator(repo *repository.GeneratedTestCodeRepository) *CppTemplateGenerator {
	return &CppTemplateGenerator{
		repo:            repo,
		functionParser:  NewFunctionParser(),
		testGenerator:   NewTestGenerator(),
		templateBuilder: NewTemplateBuilder(),
	}
}

// GenerateTemplate creates the complete template and saves it to the database
func (g *CppTemplateGenerator) GenerateTemplate(req *types.ExecutionRequest, executionID uuid.UUID) (*models.GeneratedTestCode, error) {
	startTime := time.Now()

	// Extract function name and return type using regex
	functionName, returnType, err := g.functionParser.ExtractFunctionInfo(req.Code)
	if err != nil {
		return nil, fmt.Errorf("error extracting function name: %w", err)
	}

	// Generate test code
	testCode, testCount := g.testGenerator.GenerateTestCode(req.TestCases, functionName, returnType)

	// Create complete template
	template := g.templateBuilder.BuildTemplate(req.Code, testCode)

	// Create database record
	record := &models.GeneratedTestCode{
		ExecutionID:         executionID,
		Language:            req.Language,
		GeneratorType:       "cpp_template",
		TestCode:            template,
		ChallengeID:         req.CodeVersionID.String(),
		TestCasesCount:      testCount,
		HasCustomValidation: g.templateBuilder.HasCustomValidation(req.TestCases),
		GenerationTimeMS:    time.Since(startTime).Milliseconds(),
		CodeSizeBytes:       len(template),
	}

	// Save to database
	if err := g.repo.Create(record); err != nil {
		return nil, fmt.Errorf("error saving template to database: %w", err)
	}

	return record, nil
}
