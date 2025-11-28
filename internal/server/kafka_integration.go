package server

import (
	"code-runner/internal/kafka"

	"gorm.io/gorm"
)

// Server represents the gRPC server with Kafka integration
type ServerWithKafka struct {
	// Add this field to your existing Server struct
	db          *gorm.DB
	kafkaClient *kafka.KafkaClient
}

// Example of how to integrate Kafka in your server:
//
// 1. Update your Server struct in internal/server/server_core.go:
//
// type Server struct {
//     pb.UnimplementedCodeRunnerServiceServer
//     db          *gorm.DB
//     kafkaClient *kafka.KafkaClient  // Add this field
// }
//
// 2. Update StartServer function in internal/server/server_core.go:
//
// func StartServer(port string, db *gorm.DB, kafkaClient *kafka.KafkaClient) error {
//     lis, err := net.Listen("tcp", ":"+port)
//     if err != nil {
//         return fmt.Errorf("failed to listen: %w", err)
//     }
//
//     grpcServer := grpc.NewServer(
//         grpc.MaxRecvMsgSize(8*1024*1024),
//         grpc.MaxSendMsgSize(8*1024*1024),
//     )
//
//     server := &Server{
//         db:          db,
//         kafkaClient: kafkaClient,  // Initialize Kafka client
//     }
//
//     pb.RegisterCodeRunnerServiceServer(grpcServer, server)
//
//     log.Printf("✅ gRPC server listening on port %s", port)
//     return grpcServer.Serve(lis)
// }
//
// 3. Update main.go to pass kafkaClient:
//
// if err := server.StartServer(port, database.GetDB(), kafkaClient); err != nil {
//     log.Fatalf("Failed to start server: %v", err)
// }
//
// 4. Use Kafka in your handlers (example in ExecuteCode):
//
// func (s *Server) ExecuteCode(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
//     // Your existing execution logic...
//     result, err := executeCodeLogic(req)
//     if err != nil {
//         return nil, err
//     }
//
//     // Publish event to Kafka after successful execution
//     if s.kafkaClient != nil {
//         event := &kafka.ChallengeCompletedEvent{
//             ChallengeID:   req.ChallengeId,
//             UserID:        req.StudentId,
//             ExecutionID:   result.ExecutionId,
//             Status:        "completed",
//             Score:         calculateScore(result),
//             TotalTests:    len(result.Tests),
//             PassedTests:   countPassedTests(result),
//             ExecutionTime: result.ExecutionTimeMs,
//         }
//
//         err = s.kafkaClient.PublishChallengeCompleted(ctx, event)
//         if err != nil {
//             // Log error but don't fail the request
//             log.Printf("⚠️  Warning: Failed to publish Kafka event: %v", err)
//         } else {
//             log.Printf("✅ Event published to Kafka for challenge %s", req.ChallengeId)
//         }
//     }
//
//     return result, nil
// }
//
// 5. Alternative: Publish execution events
//
// func (s *Server) ExecuteCode(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
//     // ... execution logic ...
//
//     // Publish general execution event
//     if s.kafkaClient != nil {
//         event := &kafka.CodeExecutionEvent{
//             ExecutionID: executionID,
//             Language:    req.Language,
//             Status:      "completed",
//             Output:      result.Output,
//             Metadata: map[string]interface{}{
//                 "challengeId": req.ChallengeId,
//                 "userId":      req.StudentId,
//                 "memoryUsed":  result.MemoryUsed,
//             },
//         }
//
//         err = s.kafkaClient.PublishCodeExecution(ctx, event)
//         if err != nil {
//             log.Printf("⚠️  Warning: Failed to publish execution event: %v", err)
//         }
//     }
//
//     return result, nil
// }
