package main

import (
	"context"
	"log"
	"time"

	"code-runner/env"
	"code-runner/internal/kafka"
)

// ExampleKafkaIntegration demonstrates how to integrate Kafka with the gRPC server
func ExampleKafkaIntegration() {
	// Load configuration
	config, err := env.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize Kafka client
	kafkaClient, err := kafka.NewKafkaClient(&config.Kafka)
	if err != nil {
		log.Fatalf("Failed to initialize Kafka client: %v", err)
	}
	defer kafkaClient.Close()

	// Example 1: Publish challenge completed event
	publishChallengeCompletedExample(kafkaClient)

	// Example 2: Publish code execution event
	publishCodeExecutionExample(kafkaClient)

	// Example 3: Consume messages (in a separate goroutine)
	go consumeMessagesExample(kafkaClient)

	// Wait for a while to see the consumer in action
	time.Sleep(10 * time.Second)
}

func publishChallengeCompletedExample(client *kafka.KafkaClient) {
	ctx := context.Background()

	event := &kafka.ChallengeCompletedEvent{
		ChallengeID:   "challenge-123",
		UserID:        "user-456",
		ExecutionID:   "exec-789",
		Status:        "success",
		Score:         100,
		TotalTests:    10,
		PassedTests:   10,
		ExecutionTime: 1234,
	}

	err := client.PublishChallengeCompleted(ctx, event)
	if err != nil {
		log.Printf("❌ Failed to publish challenge completed event: %v", err)
	} else {
		log.Printf("✅ Challenge completed event published successfully")
	}
}

func publishCodeExecutionExample(client *kafka.KafkaClient) {
	ctx := context.Background()

	event := &kafka.CodeExecutionEvent{
		ExecutionID: "exec-999",
		Language:    "cpp",
		Status:      "completed",
		Output:      "Test passed successfully",
		Metadata: map[string]interface{}{
			"memoryUsed": "15MB",
			"cpuTime":    "250ms",
		},
	}

	err := client.PublishCodeExecution(ctx, event)
	if err != nil {
		log.Printf("❌ Failed to publish code execution event: %v", err)
	} else {
		log.Printf("✅ Code execution event published successfully")
	}
}

func consumeMessagesExample(client *kafka.KafkaClient) {
	// Initialize consumer for a specific topic
	topic := "challenge.completed"
	err := client.InitConsumer(topic)
	if err != nil {
		log.Printf("❌ Failed to initialize consumer: %v", err)
		return
	}

	// Define message handler
	handler := func(msg interface{}) error {
		// Type assertion for kafka.Message
		log.Printf("📨 Received message: %+v", msg)
		// Process the message here
		return nil
	}

	// Start consuming (this blocks)
	ctx := context.Background()
	// Note: In real application, you should pass this handler correctly
	// This is just an example
	log.Printf("🎧 Consumer started (example mode)")
	_ = handler
	_ = ctx
}

// IntegrateKafkaWithServer shows how to integrate Kafka client with the gRPC server
// This function should be called in your server initialization
func IntegrateKafkaWithServer() {
	// In your server struct, add:
	// type Server struct {
	//     pb.UnimplementedCodeRunnerServiceServer
	//     db          *gorm.DB
	//     kafkaClient *kafka.KafkaClient  // Add this field
	// }

	// When executing code, publish events:
	// func (s *Server) ExecuteCode(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	//     // ... execute code logic ...
	//
	//     // Publish event after successful execution
	//     event := &kafka.CodeExecutionEvent{
	//         ExecutionID: executionID,
	//         Language:    req.Language,
	//         Status:      "completed",
	//         Output:      result.Output,
	//     }
	//
	//     if err := s.kafkaClient.PublishCodeExecution(ctx, event); err != nil {
	//         log.Printf("Warning: Failed to publish execution event: %v", err)
	//         // Don't fail the request if Kafka publishing fails
	//     }
	//
	//     return response, nil
	// }

	log.Printf("💡 See this file for integration examples")
}
