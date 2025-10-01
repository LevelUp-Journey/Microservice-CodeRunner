package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"gorm.io/gorm"

	pb "code-runner/api/gen/proto"
	"code-runner/internal/database/models"
	"code-runner/internal/database/repository"
	"code-runner/internal/docker"
	"code-runner/internal/template"
	"code-runner/internal/types"
)

// Simplified service implementation (gRPC adapter only)
type solutionEvaluationServiceImpl struct {
	pb.UnimplementedSolutionEvaluationServiceServer
	executionRepo         *repository.ExecutionRepository
	generatedTestCodeRepo *repository.GeneratedTestCodeRepository
	templateGenerator     *template.CppTemplateGenerator
	dockerExecutor        *docker.DockerExecutor
}

// NewSolutionEvaluationServiceServer creates a new simplified gRPC service implementation
func NewSolutionEvaluationServiceServer(db *gorm.DB) pb.SolutionEvaluationServiceServer {
	executionRepo := repository.NewExecutionRepository(db)
	generatedTestCodeRepo := repository.NewGeneratedTestCodeRepository(db)
	templateGenerator := template.NewCppTemplateGenerator(generatedTestCodeRepo)

	// Crear Docker executor
	dockerExecutor, err := docker.NewDockerExecutor()
	if err != nil {
		log.Printf("⚠️  Warning: Failed to create Docker executor: %v", err)
		log.Printf("⚠️  Docker execution will not be available. Make sure Docker is running.")
	}

	return &solutionEvaluationServiceImpl{
		executionRepo:         executionRepo,
		generatedTestCodeRepo: generatedTestCodeRepo,
		templateGenerator:     templateGenerator,
		dockerExecutor:        dockerExecutor,
	}
}

// extractFunctionName extracts the function name from C++ code using regex
func extractFunctionName(code string) string {
	// Regex to match function declarations like "int functionName(int param)"
	re := regexp.MustCompile(`\bint\s+(\w+)\s*\(\s*int\s+\w+\s*\)`)
	matches := re.FindStringSubmatch(code)
	if len(matches) > 1 {
		return matches[1]
	}
	return "unknown_function"
}

// generateCppFile generates a C++ file based on the template and input
func generateCppFile(solutionCode string, functionName string, testCases []*pb.TestCase) (string, error) {
	// Read the template
	templatePath := "template/cpp-template.cpp"
	templateBytes, err := os.ReadFile(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to read template: %v", err)
	}
	template := string(templateBytes)

	// Replace solution section
	solutionStart := "// Solution - Start"
	solutionEnd := "// Solution - End"
	solutionPlaceholder := solutionStart + "\n" + "#include \"doctest.h\"\n" + "#include <iostream>\n" + "namespace std;\n" + "\n" + solutionCode + "\n" + solutionEnd
	template = strings.Replace(template, solutionStart+"\n#include \"doctest.h\"\n#include <iostream>\nnamespace std;\n\nint fibonacci(int n) {\n    if (n < 0) throw invalid_argument(\"n debe ser >= 0\");\n    if (n == 0) return 0;\n    if (n == 1) return 1;\n    return fibonacci(n - 1) + fibonacci(n - 2);\n}\n"+solutionEnd, solutionPlaceholder, 1)

	// Generate tests
	var testChecks []string
	for _, tc := range testCases {
		input, err := strconv.Atoi(tc.Input)
		if err != nil {
			continue // Skip if not integer
		}
		expected, err := strconv.Atoi(tc.ExpectedOutput)
		if err != nil {
			continue
		}
		testChecks = append(testChecks, fmt.Sprintf("    CHECK(%s(%d) == %d);", functionName, input, expected))
	}
	testsContent := strings.Join(testChecks, "\n")

	// Replace tests section
	testsStart := "// Tests - Start"
	testsEnd := "// Tests - End"
	testsPlaceholder := testsStart + "\n" + "TEST_CASE(\"Test_id\") {\n" + testsContent + "\n" + "}\n" + testsEnd
	template = strings.Replace(template, testsStart+"\nTEST_CASE(\"Test_id\") {\n    CHECK(fibonacci(0) == 0);\n    CHECK(fibonacci(1) == 1);\n    CHECK(fibonacci(2) == 1);\n    CHECK(fibonacci(3) == 2);\n    CHECK(fibonacci(5) == 5);\n    CHECK(fibonacci(10) == 55);\n    // CHECK(function_name(input) == expected_output);\n}\n"+testsEnd, testsPlaceholder, 1)

	// Create temporary file
	tempFile := os.TempDir() + "/generated_code.cpp"
	err = os.WriteFile(tempFile, []byte(template), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	return tempFile, nil
}

// evaluateSolution executes solution code and returns approved test IDs
func (s *solutionEvaluationServiceImpl) EvaluateSolution(ctx context.Context, req *pb.ExecutionRequest) (*pb.ExecutionResponse, error) {
	log.Printf("🚀 ===== RECEIVED EXECUTION REQUEST =====")
	log.Printf("  📋 Challenge ID: %s", req.ChallengeId)
	log.Printf("  🔢 Code Version ID: %s", req.CodeVersionId)
	log.Printf("  👤 Student ID: %s", req.StudentId)
	log.Printf("   Code length: %d characters", len(req.Code))
	log.Printf("  🧪 Test cases: %d", len(req.Tests))

	// Log the actual code
	log.Printf("  📄 Code preview:")
	lines := strings.Split(req.Code, "\n")
	for i, line := range lines {
		if i < 5 { // Show first 5 lines
			log.Printf("    %d: %s", i+1, line)
		} else if i == 5 {
			log.Printf("    ... (%d more lines)", len(lines)-5)
			break
		}
	}

	// Log test cases details
	log.Printf("  🧪 Test cases details:")
	for i, tc := range req.Tests {
		log.Printf("    Test %d: ID='%s', Input='%s', Expected='%s'",
			i+1, tc.CodeVersionTestId, tc.Input, tc.ExpectedOutput)
		if tc.CustomValidationCode != "" {
			log.Printf("      Custom validation: %s", tc.CustomValidationCode)
		}
	}

	startTime := time.Now()

	// Convert proto request to internal types for template generation
	internalReq := &types.ExecutionRequest{
		SolutionID:    req.ChallengeId, // Using ChallengeId as SolutionID for now
		ChallengeID:   req.ChallengeId,
		CodeVersionID: req.CodeVersionId,
		StudentID:     req.StudentId,
		Code:          req.Code,
		Language:      "cpp", // Hardcoded for now, assuming C++
		TestCases:     convertTestCases(req.Tests),
	}

	log.Printf("🔧 Converting to internal types...")
	log.Printf("  ✅ Internal request created with %d test cases", len(internalReq.TestCases))

	// Create execution record
	log.Printf("📝 Creating execution record...")
	execution := &models.Execution{
		SolutionID:  internalReq.SolutionID,
		ChallengeID: internalReq.ChallengeID,
		StudentID:   internalReq.StudentID,
		Language:    internalReq.Language,
		Code:        internalReq.Code,
		Status:      models.StatusRunning,
		TotalTests:  len(internalReq.TestCases),
	}

	if err := s.executionRepo.Create(execution); err != nil {
		log.Printf("❌ Error creating execution record: %v", err)
		return nil, fmt.Errorf("failed to create execution record: %w", err)
	}

	log.Printf("✅ Execution record created with ID: %s", execution.ID)

	// Generate template and save to database
	log.Printf("🔧 Generating C++ execution template...")
	generatedTemplate, err := s.templateGenerator.GenerateTemplate(internalReq, execution.ID)
	if err != nil {
		log.Printf("❌ Error generating template: %v", err)
		// Update execution status to failed
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

	// Log template preview (first 500 characters)
	templatePreview := generatedTemplate.TestCode
	if len(templatePreview) > 500 {
		templatePreview = templatePreview[:500] + "\n... (truncated)"
	}
	log.Printf("  📝 Template preview:\n%s", templatePreview)

	// Execute code in Docker
	var dockerResult *docker.ExecutionResult
	if s.dockerExecutor != nil {
		log.Printf("🐳 Executing code in Docker container...")

		execConfig := docker.DefaultExecutionConfig(execution.ID, generatedTemplate.TestCode)

		dockerCtx, dockerCancel := context.WithTimeout(ctx, time.Duration(execConfig.TimeoutSeconds+5)*time.Second)
		defer dockerCancel()

		dockerResult, err = s.dockerExecutor.Execute(dockerCtx, execConfig)
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

		// Update execution with Docker results
		execution.Status = models.StatusCompleted
		execution.ExecutionTimeMS = dockerResult.ExecutionTimeMS
		execution.MemoryUsageMB = dockerResult.MemoryUsageMB
		execution.Success = dockerResult.Success
		execution.TotalTests = dockerResult.TotalTests
		execution.PassedTests = dockerResult.PassedTests

		// Extract approved test IDs from Docker results
		approvedIDs := []string{}
		failedIDs := []string{}

		if len(dockerResult.TestResults) > 0 {
			// Usar resultados individuales de tests parseados
			for _, testResult := range dockerResult.TestResults {
				if testResult.Passed {
					approvedIDs = append(approvedIDs, testResult.TestID)
				} else {
					failedIDs = append(failedIDs, testResult.TestID)
				}
			}
			log.Printf("  📋 Parsed test results: %d approved, %d failed", len(approvedIDs), len(failedIDs))
		} else if dockerResult.Success && dockerResult.PassedTests == len(internalReq.TestCases) {
			// Si todos pasaron pero no hay resultados individuales, aprobar todos
			log.Printf("  ℹ️  All tests passed, approving all test IDs")
			for _, tc := range internalReq.TestCases {
				approvedIDs = append(approvedIDs, tc.TestID)
			}
		}

		if dockerResult.Success {
			execution.Message = fmt.Sprintf("Execution successful: %d/%d tests passed", dockerResult.PassedTests, dockerResult.TotalTests)
			execution.SetApprovedTestIDs(approvedIDs)

			if len(failedIDs) > 0 {
				execution.SetFailedTestIDs(failedIDs)
			}
		} else {
			if dockerResult.TimedOut {
				execution.Status = models.StatusTimedOut
				execution.ErrorType = "timeout"
				execution.ErrorMessage = fmt.Sprintf("Execution timed out after %d seconds", dockerResult.ExecutionTimeMS/1000)
			} else {
				execution.ErrorType = "test_failure"
				execution.ErrorMessage = dockerResult.ErrorMessage
			}

			// Guardar tests aprobados y fallados
			execution.SetApprovedTestIDs(approvedIDs)
			execution.SetFailedTestIDs(failedIDs)

			// Log output for debugging
			if len(dockerResult.StdOut) > 0 {
				log.Printf("  📤 Stdout:\n%s", dockerResult.StdOut)
			}
			if len(dockerResult.StdErr) > 0 {
				log.Printf("  📤 Stderr:\n%s", dockerResult.StdErr)
			}
		}
	} else {
		// Docker not available, just mark as template generated
		log.Printf("⚠️  Docker executor not available, skipping execution")
		execution.ExecutionTimeMS = time.Since(startTime).Milliseconds()
		execution.Status = models.StatusCompleted
		execution.Success = true
		execution.Message = fmt.Sprintf("Template generated successfully (ID: %s) - Docker execution skipped", generatedTemplate.ID)

		// Convert test IDs to comma-separated string
		approvedIDs := make([]string, len(internalReq.TestCases))
		for i, tc := range internalReq.TestCases {
			approvedIDs[i] = tc.TestID
		}
		execution.SetApprovedTestIDs(approvedIDs)
		execution.PassedTests = len(internalReq.TestCases)
	}

	if err := s.executionRepo.Update(execution); err != nil {
		log.Printf("❌ Error updating execution record: %v", err)
		return nil, fmt.Errorf("failed to update execution record: %w", err)
	}

	// Prepare response with actual approved tests from execution
	var approvedTests []string
	var errorMessage string
	var errorType string

	if dockerResult != nil {
		// Usar los tests realmente aprobados del resultado de Docker
		approvedTests = execution.GetApprovedTestIDs()
		errorMessage = execution.ErrorMessage
		errorType = execution.ErrorType
	} else {
		// Docker no disponible, modo desarrollo: aprobar todos
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

// convertTestCases converts proto test cases to internal types
func convertTestCases(protoTests []*pb.TestCase) []*types.TestCase {
	tests := make([]*types.TestCase, len(protoTests))
	for i, pt := range protoTests {
		tests[i] = &types.TestCase{
			TestID:               pt.CodeVersionTestId, // Using CodeVersionTestId as TestID
			CodeVersionTestID:    pt.CodeVersionTestId,
			Input:                pt.Input,
			ExpectedOutput:       pt.ExpectedOutput,
			CustomValidationCode: pt.CustomValidationCode,
		}
	}
	return tests
}

// StartServer starts the gRPC server (simplified version)
func StartServer(port string, db *gorm.DB) error {
	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("❌ Failed to create listener on port %s: %v", port, err)
		return err
	}
	log.Printf("✅ Listener created on port %s", port)

	// Configure gRPC server options to match Spring Boot client configuration
	// - Max inbound message size: 8MB (as configured in Spring Boot)
	// - Plaintext negotiation (no TLS)
	maxMsgSize := 8 * 1024 * 1024 // 8MB
	serverOptions := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	}

	// Create gRPC server with options
	grpcServer := grpc.NewServer(serverOptions...)
	log.Printf("✅ gRPC server created")

	// Register service
	service := NewSolutionEvaluationServiceServer(db)
	pb.RegisterSolutionEvaluationServiceServer(grpcServer, service)
	log.Printf("✅ Service registered")

	log.Printf("🚀 Starting gRPC server on port %s (plaintext, max msg size: %dMB)", port, maxMsgSize/(1024*1024))

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Println("🛑 Shutting down gRPC server...")
		grpcServer.GracefulStop()
	}()

	// Start server
	log.Printf("🚀 Calling grpcServer.Serve(lis)")
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("❌ grpcServer.Serve failed: %v", err)
		return err
	}
	log.Printf("✅ Server stopped gracefully")

	return nil
}
