package adapters

import (
	"context"
	"fmt"

	"code-runner/internal/pipeline"
	"code-runner/internal/steps"
)

// DockerExecutionAdapter integrates Docker execution with the existing gRPC server
type DockerExecutionAdapter struct {
	logger pipeline.Logger
}

// NewDockerExecutionAdapter creates a new adapter
func NewDockerExecutionAdapter(logger pipeline.Logger) *DockerExecutionAdapter {
	return &DockerExecutionAdapter{
		logger: logger,
	}
}

// ExecuteWithDocker executes code using Docker-based pipeline
// This method can be called from the existing gRPC server implementation
func (dea *DockerExecutionAdapter) ExecuteWithDocker(ctx context.Context, data *pipeline.ExecutionData) error {
	dea.logger.Info(ctx, "Starting Docker-based execution", map[string]interface{}{
		"execution_id": data.ExecutionID,
		"language":     data.Language,
		"challenge_id": data.ChallengeID,
	})

	// Create pipeline with Docker execution steps
	config := &pipeline.PipelineConfig{
		ExecutionID: data.ExecutionID,
		Logger:      dea.logger,
	}

	p := pipeline.NewPipeline(config)

	// Add pipeline steps for Docker execution
	p.AddStep(steps.NewValidationStep())
	p.AddStep(steps.NewTestFetchingStep(""))            // Test API URL from config
	p.AddStep(steps.NewDockerExecutionStep(dea.logger)) // Use Docker instead of local execution
	p.AddStep(steps.NewCleanupStep())

	// Execute the pipeline
	err := p.Execute(ctx, data)
	if err != nil {
		dea.logger.Error(ctx, "Docker execution pipeline failed", err, map[string]interface{}{
			"execution_id": data.ExecutionID,
		})
		return fmt.Errorf("Docker execution failed: %w", err)
	}

	dea.logger.Info(ctx, "Docker execution completed successfully", map[string]interface{}{
		"execution_id":   data.ExecutionID,
		"success":        data.Success,
		"tests_passed":   len(data.ApprovedTestIDs),
		"execution_time": data.ExecutionTimeMS,
	})

	return nil
}

// GetApprovedTestIDs extracts approved test IDs from execution data
// This matches the proto response format
func (dea *DockerExecutionAdapter) GetApprovedTestIDs(data *pipeline.ExecutionData) []string {
	var approvedTestIds []string

	for _, result := range data.TestResults {
		if result.Passed {
			approvedTestIds = append(approvedTestIds, result.TestID)
		}
	}

	return approvedTestIds
}

// ValidateLanguageSupport checks if language is supported by Docker execution
func (dea *DockerExecutionAdapter) ValidateLanguageSupport(language string) error {
	supportedLanguages := []string{"cpp", "python", "javascript", "java", "go"}

	for _, supported := range supportedLanguages {
		if language == supported {
			return nil
		}
	}

	return fmt.Errorf("language '%s' not supported for Docker execution. Supported: %v", language, supportedLanguages)
}

// PrepareExecutionData ensures execution data is ready for Docker execution
func (dea *DockerExecutionAdapter) PrepareExecutionData(solutionID, challengeID, studentID, code, language string) *pipeline.ExecutionData {
	return &pipeline.ExecutionData{
		SolutionID:  solutionID,
		ChallengeID: challengeID,
		StudentID:   studentID,
		Code:        code,
		Language:    language,
		Config: &pipeline.ExecutionConfig{
			TimeoutSeconds:       30,
			MemoryLimitMB:        512,
			EnableNetwork:        false,
			EnvironmentVariables: make(map[string]string),
			DebugMode:            false,
		},
		Status:          pipeline.ExecutionStatusPending,
		Metadata:        make(map[string]string),
		TestResults:     make([]*pipeline.TestResult, 0),
		ApprovedTestIDs: make([]string, 0),
		CompletedSteps:  make([]pipeline.StepInfo, 0),
	}
}
