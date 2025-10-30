package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"code-runner/internal/database/models"
	"code-runner/internal/docker"
	"code-runner/internal/types"
)

// executeInDocker ejecuta el código en un contenedor Docker
func (s *solutionEvaluationServiceImpl) executeInDocker(ctx context.Context, execution *models.Execution, generatedTemplate *models.GeneratedTestCode) (*docker.ExecutionResult, error) {
	if s.dockerExecutor == nil {
		log.Printf("⚠️  Docker executor not available, skipping execution")
		execution.ExecutionTimeMS = 0
		execution.Status = models.StatusCompleted
		execution.Success = true
		execution.Message = "Template generated successfully - Docker execution skipped"
		return nil, nil
	}

	log.Printf("🐳 Executing code in Docker container...")

	execConfig := docker.DefaultExecutionConfig(execution.ID, generatedTemplate.TestCode)

	dockerCtx, dockerCancel := context.WithTimeout(ctx, time.Duration(execConfig.TimeoutSeconds+5)*time.Second)
	defer dockerCancel()

	dockerResult, err := s.dockerExecutor.Execute(dockerCtx, execConfig)
	if err != nil {
		log.Printf("❌ Docker execution error: %v", err)
		execution.Status = models.StatusFailed
		execution.ErrorMessage = fmt.Sprintf("Docker execution failed: %v", err)
		execution.ErrorType = "docker_error"
		s.executionRepo.Update(execution)
		return nil, fmt.Errorf("failed to execute in Docker: %w", err)
	}

	log.Printf("✅ Docker execution completed")
	log.Printf("  ⏱️  Execution time: %d ms", dockerResult.ExecutionTimeMS)
	log.Printf("  📊 Exit code: %d", dockerResult.ExitCode)
	log.Printf("  🧪 Tests: %d/%d passed", dockerResult.PassedTests, dockerResult.TotalTests)

	return dockerResult, nil
}

// processResults procesa los resultados de la ejecución Docker
func (s *solutionEvaluationServiceImpl) processResults(execution *models.Execution, dockerResult *docker.ExecutionResult, req *types.ExecutionRequest) *models.Execution {
	if dockerResult == nil {
		// Docker not available - development mode
		approvedIDs := make([]string, len(req.TestCases))
		for i, tc := range req.TestCases {
			approvedIDs[i] = tc.TestID.String()
		}
		execution.SetApprovedTestIDs(approvedIDs)
		execution.PassedTests = len(req.TestCases)
		return execution
	}

	// Update execution with Docker results
	execution.Status = models.StatusCompleted
	execution.ExecutionTimeMS = dockerResult.ExecutionTimeMS
	execution.MemoryUsageMB = dockerResult.MemoryUsageMB
	execution.Success = dockerResult.Success
	execution.TotalTests = dockerResult.TotalTests
	execution.PassedTests = dockerResult.PassedTests

	// Extract test IDs
	approvedIDs, failedIDs := s.extractTestIDs(dockerResult, req)

	log.Printf("  📋 Parsed test results: %d approved, %d failed", len(approvedIDs), len(failedIDs))

	// Handle success case
	if dockerResult.Success {
		execution.Message = fmt.Sprintf("Execution successful: %d/%d tests passed", dockerResult.PassedTests, dockerResult.TotalTests)
		execution.SetApprovedTestIDs(approvedIDs)
		if len(failedIDs) > 0 {
			execution.SetFailedTestIDs(failedIDs)
		}
		return execution
	}

	// Handle failure case
	if dockerResult.TimedOut {
		execution.Status = models.StatusTimedOut
		execution.ErrorType = "timeout"
		execution.ErrorMessage = fmt.Sprintf("Execution timed out after %d seconds", dockerResult.ExecutionTimeMS/1000)
	} else {
		execution.ErrorType = "test_failure"
		execution.ErrorMessage = dockerResult.ErrorMessage
	}

	execution.SetApprovedTestIDs(approvedIDs)
	execution.SetFailedTestIDs(failedIDs)

	// Log output for debugging
	if len(dockerResult.StdOut) > 0 {
		log.Printf("  📤 Stdout:\n%s", dockerResult.StdOut)
	}
	if len(dockerResult.StdErr) > 0 {
		log.Printf("  📤 Stderr:\n%s", dockerResult.StdErr)
	}

	return execution
}

// extractTestIDs extrae los IDs de tests aprobados y fallidos
func (s *solutionEvaluationServiceImpl) extractTestIDs(dockerResult *docker.ExecutionResult, req *types.ExecutionRequest) ([]string, []string) {
	approvedIDs := []string{}
	failedIDs := []string{}

	if len(dockerResult.TestResults) > 0 {
		for _, testResult := range dockerResult.TestResults {
			if testResult.Passed {
				approvedIDs = append(approvedIDs, testResult.TestID)
			} else {
				failedIDs = append(failedIDs, testResult.TestID)
			}
		}
	} else if dockerResult.Success && dockerResult.PassedTests == len(req.TestCases) {
		// All tests passed but no individual results
		log.Printf("  ℹ️  All tests passed, approving all test IDs")
		for _, tc := range req.TestCases {
			approvedIDs = append(approvedIDs, tc.TestID.String())
		}
	}

	return approvedIDs, failedIDs
}
