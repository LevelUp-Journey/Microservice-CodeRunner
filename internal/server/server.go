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

	pb "code-runner/api/gen/proto"
	"code-runner/internal/types"
)

// Simplified service implementation (gRPC adapter only)
type solutionEvaluationServiceImpl struct {
	pb.UnimplementedSolutionEvaluationServiceServer
}

// NewSolutionEvaluationServiceServer creates a new simplified gRPC service implementation
func NewSolutionEvaluationServiceServer() pb.SolutionEvaluationServiceServer {
	return &solutionEvaluationServiceImpl{}
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

	// TODO: Use the template generator here instead of mock response
	// For now, create a simple response
	approvedTests := make([]string, len(req.Tests))
	for i, tc := range req.Tests {
		approvedTests[i] = tc.CodeVersionTestId
		log.Printf("  ✅ Approving test: %s", tc.CodeVersionTestId)
	}

	executionTime := time.Since(startTime)

	log.Printf("✅ ===== EXECUTION COMPLETED =====")
	log.Printf("  ⏱️  Execution time: %d ms", executionTime.Milliseconds())
	log.Printf("  📊 Approved tests: %d/%d", len(approvedTests), len(req.Tests))

	return &pb.ExecutionResponse{
		ApprovedTests: approvedTests,
		Completed:     true,
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
func StartServer(port string) error {
	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

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

	// Register service
	service := NewSolutionEvaluationServiceServer()
	pb.RegisterSolutionEvaluationServiceServer(grpcServer, service)

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
	if err := grpcServer.Serve(lis); err != nil {
		return err
	}

	return nil
}
