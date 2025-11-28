package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"code-runner/internal/database/models"
	"code-runner/internal/docker"
	"code-runner/internal/kafka"
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
		execution.Message = "Execution timed out"
	} else {
		// Use error type from dockerResult (compilation_error, runtime_error, etc.)
		execution.ErrorType = dockerResult.ErrorType
		execution.ErrorMessage = dockerResult.ErrorMessage

		// Set appropriate message based on error type
		if dockerResult.ErrorType == "compilation_error" {
			execution.Message = "Compilation failed"
		} else if dockerResult.ErrorType == "runtime_error" {
			execution.Message = "Runtime error occurred"
		} else {
			execution.Message = fmt.Sprintf("Execution failed: %d/%d tests passed", dockerResult.PassedTests, dockerResult.TotalTests)
		}
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

// publishMetricsToKafka publica las métricas de ejecución a Kafka
func (s *solutionEvaluationServiceImpl) publishMetricsToKafka(ctx context.Context, execution *models.Execution, dockerResult *docker.ExecutionResult, executionTimeMS int64) {
	// Si no hay cliente de Kafka, no hacer nada
	if s.kafkaClient == nil {
		log.Printf("⚠️  Kafka client not available, skipping metrics publishing")
		return
	}

	// Construir el evento de métricas
	event := &kafka.ExecutionMetricsEvent{
		ExecutionID:   execution.ID.String(),
		ChallengeID:   execution.ChallengeID,
		CodeVersionID: execution.SolutionID, // SolutionID is used as CodeVersionID
		StudentID:     execution.StudentID,
		Language:      execution.Language,
		Status:        string(execution.Status),
		Timestamp:     time.Now(),

		// Métricas de rendimiento
		ExecutionTimeMS: executionTimeMS,
		TotalTests:      execution.TotalTests,
		PassedTests:     execution.PassedTests,
		FailedTests:     execution.TotalTests - execution.PassedTests,
		Success:         execution.Success,

		// Errores
		ErrorMessage: execution.ErrorMessage,
		ErrorType:    execution.ErrorType,

		// Metadata del servidor
		ServerInstance: getServerInstance(),
	}

	// Agregar métricas de Docker si están disponibles
	if dockerResult != nil {
		event.MemoryUsageMB = dockerResult.MemoryUsageMB
		event.ExitCode = dockerResult.ExitCode

		// Agregar resultados de tests individuales
		if len(dockerResult.TestResults) > 0 {
			event.TestResults = make([]kafka.TestResultMetric, 0, len(dockerResult.TestResults))
			for _, testResult := range dockerResult.TestResults {
				event.TestResults = append(event.TestResults, kafka.TestResultMetric{
					TestID:          testResult.TestID,
					TestName:        testResult.TestName,
					Passed:          testResult.Passed,
					ExecutionTimeMS: testResult.ExecutionTimeMS,
					ErrorMessage:    testResult.ErrorMessage,
				})
			}
		}
	}

	// Publicar a Kafka de forma asíncrona
	go func() {
		publishCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.kafkaClient.PublishExecutionMetrics(publishCtx, event); err != nil {
			log.Printf("⚠️  Failed to publish metrics to Kafka: %v", err)
		} else {
			log.Printf("✅ Metrics published to Kafka for execution %s", execution.ID)
		}
	}()
}

// getServerInstance obtiene el identificador de instancia del servidor
func getServerInstance() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}
