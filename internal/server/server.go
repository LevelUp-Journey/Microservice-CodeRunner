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
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "code-runner/api/gen/proto"
)

// Simplified service implementation (gRPC adapter only)
type codeExecutionServiceImpl struct {
	pb.UnimplementedCodeExecutionServiceServer
}

// NewCodeExecutionServiceServer creates a new simplified gRPC service implementation
func NewCodeExecutionServiceServer() pb.CodeExecutionServiceServer {
	return &codeExecutionServiceImpl{}
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

// ExecuteCode executes solution code and returns approved test IDs (simplified - no pipeline)
func (s *codeExecutionServiceImpl) ExecuteCode(ctx context.Context, req *pb.ExecutionRequest) (*pb.ExecutionResponse, error) {
	// Extract function name from solution code
	functionName := extractFunctionName(req.Code)

	// Generate C++ file
	filePath, err := generateCppFile(req.Code, functionName, req.TestCases)
	if err != nil {
		log.Printf("Error generating C++ file: %v", err)
		return &pb.ExecutionResponse{
			Success: false,
			Message: fmt.Sprintf("Failed to generate C++ file: %v", err),
		}, nil
	}

	log.Printf("Generated C++ file at: %s", filePath)

	// Simple response (for now, just indicate file created)
	return &pb.ExecutionResponse{
		PassedTestsId: []string{"test1"}, // Mock
		TimeTaken:     100,
		Success:       true,
		Message:       fmt.Sprintf("C++ file generated successfully at %s", filePath),
	}, nil
}

// GetExecutionStatus returns the status of an execution (simplified)
func (s *codeExecutionServiceImpl) GetExecutionStatus(ctx context.Context, req *pb.ExecutionStatusRequest) (*pb.ExecutionStatusResponse, error) {
	log.Printf("📊 Received status request for execution: %s", req.ExecutionId)

	// Return a simple completed status
	return &pb.ExecutionStatusResponse{
		ExecutionId:     req.ExecutionId,
		Status:          pb.ExecutionStatus_EXECUTION_STATUS_COMPLETED,
		Message:         "Execution completed successfully (simplified mode)",
		Success:         true,
		ApprovedTestIds: []string{"test1", "test2", "test3"},
	}, nil
}

// HealthCheck provides a simple health check endpoint
func (s *codeExecutionServiceImpl) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:    pb.HealthStatus_HEALTH_STATUS_SERVING,
		Message:   "gRPC adapter is healthy",
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

// StartServer starts the gRPC server (simplified version)
func StartServer(port string) error {
	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return err
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register service
	service := NewCodeExecutionServiceServer()
	pb.RegisterCodeExecutionServiceServer(grpcServer, service)

	log.Printf("🚀 Starting simplified gRPC server on port %s (adapter only)", port)

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
