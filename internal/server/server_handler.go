package server

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"

	pb "code-runner/api/gen/proto"
	"code-runner/internal/database/models"
	"code-runner/internal/docker"
	"code-runner/internal/types"
)

// EvaluateSolution maneja la evaluación de soluciones de código
func (s *solutionEvaluationServiceImpl) EvaluateSolution(ctx context.Context, req *pb.ExecutionRequest) (*pb.ExecutionResponse, error) {
	log.Printf("🚀 ===== RECEIVED EXECUTION REQUEST =====")
	log.Printf("  📋 Challenge ID: '%s' (len: %d)", req.ChallengeId, len(req.ChallengeId))
	log.Printf("  🔢 Code Version ID: '%s' (len: %d)", req.CodeVersionId, len(req.CodeVersionId))
	log.Printf("  👤 Student ID: '%s' (len: %d)", req.StudentId, len(req.StudentId))
	log.Printf("  💻 Code length: %d characters", len(req.Code))
	log.Printf("  🧪 Test cases: %d", len(req.Tests))

	// Log code preview
	s.logCodePreview(req.Code)

	// Log test cases
	s.logTestCases(req.Tests)

	startTime := time.Now()

	// Parse and validate UUIDs
	internalReq, err := s.parseAndValidateRequest(req)
	if err != nil {
		return nil, err
	}

	// Create execution record
	execution, err := s.createExecutionRecord(internalReq)
	if err != nil {
		return nil, err
	}

	// Generate template
	generatedTemplate, err := s.generateTemplate(internalReq, execution)
	if err != nil {
		return nil, err
	}

	// Execute in Docker
	dockerResult, err := s.executeInDocker(ctx, execution, generatedTemplate)
	if err != nil {
		return nil, err
	}

	// Process results
	execution = s.processResults(execution, dockerResult, internalReq)

	// Update execution record
	if err := s.executionRepo.Update(execution); err != nil {
		log.Printf("❌ Error updating execution record: %v", err)
		return nil, fmt.Errorf("failed to update execution record: %w", err)
	}

	// Calculate total execution time
	executionTime := time.Since(startTime)

	// Publish metrics to Kafka
	s.publishMetricsToKafka(ctx, execution, dockerResult, executionTime.Milliseconds())

	// Build response
	return s.buildResponse(req, execution, dockerResult, startTime)
}

// logCodePreview registra un preview del código
func (s *solutionEvaluationServiceImpl) logCodePreview(code string) {
	log.Printf("  📄 Code preview:")
	lines := strings.Split(code, "\n")
	for i, line := range lines {
		if i < 5 {
			log.Printf("    %d: %s", i+1, line)
		} else if i == 5 {
			log.Printf("    ... (%d more lines)", len(lines)-5)
			break
		}
	}
}

// logTestCases registra los detalles de los test cases
func (s *solutionEvaluationServiceImpl) logTestCases(tests []*pb.TestCase) {
	log.Printf("  🧪 Test cases details:")
	for i, tc := range tests {
		log.Printf("    Test %d: ID='%s', Input='%s', Expected='%s'",
			i+1, tc.CodeVersionTestId, tc.Input, tc.ExpectedOutput)
		if tc.CustomValidationCode != "" {
			log.Printf("      Custom validation: %s", tc.CustomValidationCode)
		}
	}
}

// parseAndValidateRequest valida y parsea el request
func (s *solutionEvaluationServiceImpl) parseAndValidateRequest(req *pb.ExecutionRequest) (*types.ExecutionRequest, error) {
	// Helper function to parse UUID
	parseUUID := func(id string, fieldName string) (uuid.UUID, error) {
		// If ID is empty, return error
		if id == "" {
			return uuid.Nil, fmt.Errorf("%s is required and cannot be empty", fieldName)
		}

		// Try to parse the UUID
		parsed, err := uuid.Parse(id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid %s format: %s", fieldName, id)
		}

		return parsed, nil
	}

	challengeID, err := parseUUID(req.ChallengeId, "ChallengeId")
	if err != nil {
		return nil, err
	}

	codeVersionID, err := parseUUID(req.CodeVersionId, "CodeVersionId")
	if err != nil {
		return nil, err
	}

	studentID, err := parseUUID(req.StudentId, "StudentId")
	if err != nil {
		return nil, err
	}

	// Use default language (C++) for now
	language := "cpp"

	internalReq := &types.ExecutionRequest{
		SolutionID:    challengeID,
		ChallengeID:   challengeID,
		CodeVersionID: codeVersionID,
		StudentID:     studentID,
		Code:          req.Code,
		Language:      language,
		TestCases:     convertTestCases(req.Tests),
	}

	log.Printf("🔧 Converting to internal types...")
	log.Printf("  ✅ Internal request created with %d test cases", len(internalReq.TestCases))
	log.Printf("  🆔 Generated IDs - Challenge: %s, CodeVersion: %s, Student: %s",
		challengeID.String(), codeVersionID.String(), studentID.String())

	return internalReq, nil
}

// createExecutionRecord crea un registro de ejecución en la base de datos
func (s *solutionEvaluationServiceImpl) createExecutionRecord(req *types.ExecutionRequest) (*models.Execution, error) {
	log.Printf("📝 Creating execution record...")
	execution := &models.Execution{
		SolutionID:  req.SolutionID.String(),
		ChallengeID: req.ChallengeID.String(),
		StudentID:   req.StudentID.String(),
		Language:    req.Language,
		Code:        req.Code,
		Status:      models.StatusRunning,
		TotalTests:  len(req.TestCases),
	}

	if err := s.executionRepo.Create(execution); err != nil {
		log.Printf("❌ Error creating execution record: %v", err)
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	log.Printf("✅ Execution record created with ID: %s", execution.ID)
	return execution, nil
}

// generateTemplate genera el template de código C++
func (s *solutionEvaluationServiceImpl) generateTemplate(req *types.ExecutionRequest, execution *models.Execution) (*models.GeneratedTestCode, error) {
	log.Printf("🔧 Generating C++ execution template...")
	generatedTemplate, err := s.templateGenerator.GenerateTemplate(req, execution.ID)
	if err != nil {
		log.Printf("❌ Error generating template: %v", err)
		execution.Status = models.StatusFailed
		execution.ErrorMessage = fmt.Sprintf("Template generation failed: %v", err)
		s.executionRepo.Update(execution)
		return nil, fmt.Errorf("failed to generate template: %w", err)
	}

	log.Printf("✅ Template generated and saved to database")
	log.Printf("  📄 Template ID: %s", generatedTemplate.ID)
	log.Printf("  📏 Template size: %d bytes", generatedTemplate.CodeSizeBytes)
	log.Printf("  🧪 Test cases in template: %d", generatedTemplate.TestCasesCount)
	log.Printf("  ⏱️  Generation time: %d ms", generatedTemplate.GenerationTimeMS)

	return generatedTemplate, nil
}

// buildResponse construye la respuesta gRPC
func (s *solutionEvaluationServiceImpl) buildResponse(req *pb.ExecutionRequest, execution *models.Execution, dockerResult *docker.ExecutionResult, startTime time.Time) (*pb.ExecutionResponse, error) {
	var approvedTests []string
	var errorMessage string
	var errorType string

	if dockerResult != nil {
		approvedTests = execution.GetApprovedTestIDs()
		errorMessage = execution.ErrorMessage
		errorType = execution.ErrorType
	} else {
		approvedTests = make([]string, len(req.Tests))
		for i, tc := range req.Tests {
			approvedTests[i] = tc.CodeVersionTestId
		}
	}

	executionTime := time.Since(startTime)
	totalTests := len(req.Tests)
	passedTests := len(approvedTests)
	failedTests := totalTests - passedTests

	log.Printf("✅ ===== EXECUTION COMPLETED =====")
	log.Printf("  ⏱️  Total execution time: %d ms", executionTime.Milliseconds())
	log.Printf("  📊 Test results: %d/%d passed, %d failed", passedTests, totalTests, failedTests)
	log.Printf("  ✅ Approved test IDs: %v", approvedTests)
	log.Printf("  ✅ Success: %v", execution.Success)

	return &pb.ExecutionResponse{
		ApprovedTests:   approvedTests,
		Completed:       true,
		ExecutionTimeMs: executionTime.Milliseconds(),
		TotalTests:      int32(totalTests),
		PassedTests:     int32(passedTests),
		FailedTests:     int32(failedTests),
		Success:         execution.Success,
		Message:         execution.Message,
		ErrorMessage:    errorMessage,
		ErrorType:       errorType,
	}, nil
}

// mapProtoLanguage mapea el lenguaje del proto a internal language
func mapProtoLanguage(protoLang string) string {
	// For now, default to C++ since language field is not properly implemented
	return "cpp"
}
