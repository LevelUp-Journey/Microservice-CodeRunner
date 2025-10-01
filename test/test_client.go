package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "code-runner/api/gen/proto"
)

func main() {
	// Connect to gRPC server
	conn, err := grpc.Dial("127.0.0.1:9084", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewSolutionEvaluationServiceClient(conn)

	// Create test request
	req := &pb.ExecutionRequest{
		ChallengeId:   "test-challenge-456",
		CodeVersionId: "test-solution-123",
		StudentId:     "test-student-789",
		Code:          "#include <iostream>\nint main() { std::cout << \"Hello World\"; return 0; }",
		Tests: []*pb.TestCase{
			{
				CodeVersionTestId: "test-1",
				Input:             "",
				ExpectedOutput:    "Hello World",
			},
		},
	}

	fmt.Println("📤 Sending test request to gRPC server...")

	// Call the service
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := client.EvaluateSolution(ctx, req)
	if err != nil {
		log.Fatalf("EvaluateSolution failed: %v", err)
	}

	fmt.Printf("✅ Response received: Completed=%v\n", resp.Completed)
	fmt.Printf("📊 Approved Tests: %v\n", resp.ApprovedTests)

	// Check database for saved data
	fmt.Println("🔍 Checking database for saved execution and generated code...")
}
