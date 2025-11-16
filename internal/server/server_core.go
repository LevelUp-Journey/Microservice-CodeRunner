package server

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"gorm.io/gorm"

	pb "code-runner/api/gen/proto"
	"code-runner/internal/database/repository"
	"code-runner/internal/docker"
	"code-runner/internal/kafka"
	template "code-runner/internal/template/cpp"
)

// solutionEvaluationServiceImpl implementa el servicio gRPC
type solutionEvaluationServiceImpl struct {
	pb.UnimplementedSolutionEvaluationServiceServer
	executionRepo         *repository.ExecutionRepository
	generatedTestCodeRepo *repository.GeneratedTestCodeRepository
	templateGenerator     *template.CppTemplateGenerator
	dockerExecutor        *docker.DockerExecutor
	kafkaClient           *kafka.KafkaClient
}

// NewSolutionEvaluationServiceServer crea una nueva instancia del servicio
func NewSolutionEvaluationServiceServer(db *gorm.DB, kafkaClient *kafka.KafkaClient) pb.SolutionEvaluationServiceServer {
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
		kafkaClient:           kafkaClient,
	}
}

// StartServer inicia el servidor gRPC
func StartServer(port string, db *gorm.DB, kafkaClient *kafka.KafkaClient) error {
	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Printf("❌ Failed to create listener on port %s: %v", port, err)
		return err
	}
	log.Printf("✅ Listener created on port %s", port)

	// Configure gRPC server options
	maxMsgSize := 8 * 1024 * 1024 // 8MB
	serverOptions := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxMsgSize),
		grpc.MaxSendMsgSize(maxMsgSize),
	}

	// Create gRPC server
	grpcServer := grpc.NewServer(serverOptions...)
	log.Printf("✅ gRPC server created")

	// Register service
	service := NewSolutionEvaluationServiceServer(db, kafkaClient)
	pb.RegisterSolutionEvaluationServiceServer(grpcServer, service)
	log.Printf("✅ Service registered")

	// Register reflection so tools (grpcurl, BloomRPC, Bruno) can discover services/methods
	reflection.Register(grpcServer)
	log.Printf("✅ gRPC reflection registered")

	// Handle graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
		<-sigChan

		log.Printf("🛑 Received shutdown signal, stopping server...")
		grpcServer.GracefulStop()
		log.Printf("✅ Server stopped gracefully")
	}()

	// Start server
	log.Printf("🚀 Starting gRPC server on port %s...", port)
	if err := grpcServer.Serve(lis); err != nil {
		log.Printf("❌ Failed to serve: %v", err)
		return err
	}

	return nil
}
