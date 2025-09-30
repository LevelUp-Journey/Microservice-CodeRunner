package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "code-runner/api/proto"
	"code-runner/internal/database"
	"code-runner/internal/database/models"
	"code-runner/internal/database/repository"
	"code-runner/internal/pipeline"
	"code-runner/internal/services"
	"code-runner/internal/steps"
)

// Actual service implementation
type codeExecutionServiceImpl struct {
	pb.UnimplementedCodeExecutionServiceServer
	logger           pipeline.Logger
	challengesAPIURL string
	executions       sync.Map // Store active executions
	executionService *services.ExecutionService
}

// NewCodeExecutionServiceServer creates a new gRPC service implementation
func NewCodeExecutionServiceServer(logger pipeline.Logger, challengesAPIURL string) pb.CodeExecutionServiceServer {
	return &codeExecutionServiceImpl{
		logger:           logger,
		challengesAPIURL: challengesAPIURL,
	}
}

// NewCodeExecutionServiceServerWithDB creates a new gRPC service implementation with database support
func NewCodeExecutionServiceServerWithDB(logger pipeline.Logger, challengesAPIURL string, executionService *services.ExecutionService) pb.CodeExecutionServiceServer {
	return &codeExecutionServiceImpl{
		logger:           logger,
		challengesAPIURL: challengesAPIURL,
		executionService: executionService,
	}
}

// ExecuteCode executes solution code and returns approved test IDs
func (s *codeExecutionServiceImpl) ExecuteCode(ctx context.Context, req *pb.ExecutionRequest) (*pb.ExecutionResponse, error) {
	// Generate execution ID
	executionID := generateExecutionID()

	s.logger.Info(ctx, "Received code execution request", map[string]interface{}{
		"execution_id": executionID,
		"solution_id":  req.SolutionId,
		"challenge_id": req.ChallengeId,
		"student_id":   req.StudentId,
		"language":     req.Language,
	})

	// Create database execution record if service is available
	var dbExecution *models.Execution
	var dbExecutionID uuid.UUID
	var err error

	if s.executionService != nil {
		dbExecution, err = s.executionService.CreateExecution(
			req.SolutionId,
			req.ChallengeId,
			req.StudentId,
			req.Code,
			req.Language,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to create database execution record", err, map[string]interface{}{
				"execution_id": executionID,
			})
			// Continue without database tracking
		} else {
			dbExecutionID = dbExecution.ID
			// Start the execution in database
			if err := s.executionService.StartExecution(dbExecutionID); err != nil {
				s.logger.Error(ctx, "Failed to start database execution", err, nil)
			}
		}
	}

	// Create database event handler if available
	var eventHandler pipeline.EventHandler
	if s.executionService != nil {
		eventHandler = pipeline.NewDatabaseEventHandler(s.executionService, s.logger)
	}

	// Create pipeline
	pipelineConfig := &pipeline.PipelineConfig{
		ExecutionID:  executionID,
		Logger:       s.logger,
		EventHandler: eventHandler,
	}

	p := pipeline.NewPipeline(pipelineConfig)

	// Add pipeline steps
	if err := s.setupPipeline(p); err != nil {
		s.logger.Error(ctx, "Failed to setup pipeline", err, map[string]interface{}{
			"execution_id": executionID,
		})

		// Mark database execution as failed if available
		if s.executionService != nil && dbExecutionID != uuid.Nil {
			s.executionService.FailExecution(dbExecutionID, "Failed to setup pipeline", "setup_error")
		}

		return nil, status.Errorf(codes.Internal, "Failed to setup execution pipeline: %v", err)
	}

	// Create execution data
	execData := s.createExecutionData(req, executionID)
	execData.DatabaseExecutionID = dbExecutionID // Store DB ID for tracking

	// Store execution for status tracking
	s.executions.Store(executionID, execData)

	// Execute pipeline
	startTime := time.Now()
	if err := p.Execute(ctx, execData); err != nil {
		s.logger.Error(ctx, "Pipeline execution failed", err, map[string]interface{}{
			"execution_id": executionID,
		})

		// Mark database execution as failed if available
		if s.executionService != nil && dbExecutionID != uuid.Nil {
			s.executionService.FailExecution(dbExecutionID, fmt.Sprintf("Pipeline execution failed: %v", err), "pipeline_error")

			// Add execution logs
			s.executionService.AddLog(dbExecutionID, "error", fmt.Sprintf("Pipeline execution failed: %v", err), "pipeline")
		}

		return &pb.ExecutionResponse{
			ApprovedTestIds: []string{},
			Success:         false,
			Message:         fmt.Sprintf("Execution failed: %v", err),
			ExecutionId:     executionID,
			Metadata:        s.buildExecutionMetadata(execData),
			PipelineSteps:   s.buildPipelineSteps(execData),
		}, nil
	}

	executionDuration := time.Since(startTime).Milliseconds()

	s.logger.Info(ctx, "Pipeline execution completed", map[string]interface{}{
		"execution_id":   executionID,
		"success":        execData.Success,
		"approved_tests": len(execData.ApprovedTestIDs),
		"execution_time": execData.ExecutionTimeMS,
	})

	// Update database execution with results if available
	if s.executionService != nil && dbExecutionID != uuid.Nil {
		failedTestIDs := make([]string, 0)
		// For now, we'll assume all non-approved tests are failed
		// This could be enhanced with more detailed test tracking

		memoryUsage := 0.0 // Default value, could be enhanced with actual memory tracking

		err := s.executionService.CompleteExecution(
			dbExecutionID,
			execData.Success,
			execData.Message,
			execData.ApprovedTestIDs,
			failedTestIDs,
			executionDuration,
			memoryUsage,
		)
		if err != nil {
			s.logger.Error(ctx, "Failed to complete database execution", err, nil)
		}

		// Add completion log
		s.executionService.AddLog(dbExecutionID, "info",
			fmt.Sprintf("Execution completed successfully. Tests passed: %d", len(execData.ApprovedTestIDs)),
			"system")
	}

	// Build response
	response := &pb.ExecutionResponse{
		ApprovedTestIds: execData.ApprovedTestIDs,
		Success:         execData.Success,
		Message:         execData.Message,
		ExecutionId:     executionID,
		Metadata:        s.buildExecutionMetadata(execData),
		PipelineSteps:   s.buildPipelineSteps(execData),
	}

	return response, nil
}

// GetExecutionStatus gets the status of an execution
func (s *codeExecutionServiceImpl) GetExecutionStatus(ctx context.Context, req *pb.ExecutionStatusRequest) (*pb.ExecutionStatusResponse, error) {
	s.logger.Info(ctx, "Received execution status request", map[string]interface{}{
		"execution_id": req.ExecutionId,
	})

	// Get execution data
	execDataInterface, exists := s.executions.Load(req.ExecutionId)
	if !exists {
		return nil, status.Errorf(codes.NotFound, "Execution not found: %s", req.ExecutionId)
	}

	execData, ok := execDataInterface.(*pipeline.ExecutionData)
	if !ok {
		return nil, status.Errorf(codes.Internal, "Invalid execution data")
	}

	// Convert pipeline status to proto status
	protoStatus := s.convertExecutionStatus(execData.Status)

	response := &pb.ExecutionStatusResponse{
		ExecutionId:     req.ExecutionId,
		Status:          protoStatus,
		ApprovedTestIds: execData.ApprovedTestIDs,
		Success:         execData.Success,
		Message:         execData.Message,
		Metadata:        s.buildExecutionMetadata(execData),
		PipelineSteps:   s.buildPipelineSteps(execData),
	}

	return response, nil
}

// HealthCheck returns the health status of the service
func (s *codeExecutionServiceImpl) HealthCheck(ctx context.Context, req *pb.HealthCheckRequest) (*pb.HealthCheckResponse, error) {
	return &pb.HealthCheckResponse{
		Status:    pb.HealthStatus_HEALTH_STATUS_SERVING,
		Message:   "Code Runner service is healthy",
		Timestamp: timestamppb.New(time.Now()),
	}, nil
}

// StreamExecutionLogs streams execution logs in real-time
func (s *codeExecutionServiceImpl) StreamExecutionLogs(req *pb.StreamLogsRequest, stream pb.CodeExecutionService_StreamExecutionLogsServer) error {
	ctx := stream.Context()

	s.logger.Info(ctx, "Starting log stream", map[string]interface{}{
		"execution_id": req.ExecutionId,
	})

	// Get execution data
	execDataInterface, exists := s.executions.Load(req.ExecutionId)
	if !exists {
		return status.Errorf(codes.NotFound, "Execution not found: %s", req.ExecutionId)
	}

	execData, ok := execDataInterface.(*pipeline.ExecutionData)
	if !ok {
		return status.Errorf(codes.Internal, "Invalid execution data")
	}

	// Stream existing logs
	for _, logEntry := range execData.Logs {
		protoLogEntry := &pb.LogEntry{
			Timestamp: timestamppb.New(logEntry.Timestamp),
			Level:     s.convertLogLevel(logEntry.Level),
			Message:   logEntry.Message,
			StepName:  logEntry.StepName,
			Metadata:  logEntry.Metadata,
		}

		if err := stream.Send(protoLogEntry); err != nil {
			return err
		}
	}

	return nil
}

// setupPipeline configures the execution pipeline with all necessary steps
func (s *codeExecutionServiceImpl) setupPipeline(p pipeline.Pipeline) error {
	// Step 1: Validation
	validationStep := steps.NewValidationStep()
	if err := p.AddStep(validationStep); err != nil {
		return fmt.Errorf("failed to add validation step: %w", err)
	}

	// Step 2: Compilation
	compilationStep := steps.NewCompilationStep()
	if err := p.AddStep(compilationStep); err != nil {
		return fmt.Errorf("failed to add compilation step: %w", err)
	}

	// Step 3: Test Fetching
	testFetchingStep := steps.NewTestFetchingStep(s.challengesAPIURL)
	if err := p.AddStep(testFetchingStep); err != nil {
		return fmt.Errorf("failed to add test fetching step: %w", err)
	}

	// Step 4: Docker Execution (with database support if available)
	var executionStep pipeline.PipelineStep
	if s.executionService != nil {
		// Get the database connection from execution service
		// We need to access the underlying DB - this is a bit of a hack but works
		db := s.executionService.GetDB()
		generatedTestCodeRepo := repository.NewGeneratedTestCodeRepository(db)
		executionStep = steps.NewDockerExecutionStepWithRepo(s.logger, generatedTestCodeRepo)
	} else {
		executionStep = steps.NewDockerExecutionStep(s.logger)
	}
	if err := p.AddStep(executionStep); err != nil {
		return fmt.Errorf("failed to add docker execution step: %w", err)
	}

	// Step 5: Cleanup
	cleanupStep := steps.NewCleanupStep()
	if err := p.AddStep(cleanupStep); err != nil {
		return fmt.Errorf("failed to add cleanup step: %w", err)
	}

	return nil
}

// createExecutionData creates execution data from the request
func (s *codeExecutionServiceImpl) createExecutionData(req *pb.ExecutionRequest, executionID string) *pipeline.ExecutionData {
	// Create execution config
	config := &pipeline.ExecutionConfig{
		TimeoutSeconds:       30,
		MemoryLimitMB:        512,
		EnableNetwork:        false,
		EnvironmentVariables: make(map[string]string),
		DebugMode:            false,
	}

	// Override with request config if provided
	if req.Config != nil {
		config.TimeoutSeconds = req.Config.TimeoutSeconds
		config.MemoryLimitMB = req.Config.MemoryLimitMb
		config.EnableNetwork = req.Config.EnableNetwork
		config.EnvironmentVariables = req.Config.EnvironmentVariables
		config.DebugMode = req.Config.DebugMode
	}

	return &pipeline.ExecutionData{
		SolutionID:          req.SolutionId,
		ChallengeID:         req.ChallengeId,
		StudentID:           req.StudentId,
		Code:                req.Code,
		Language:            req.Language,
		Config:              config,
		ExecutionID:         executionID,
		DatabaseExecutionID: uuid.Nil, // Will be set after DB creation
		Status:              pipeline.ExecutionStatusPending,
		ApprovedTestIDs:     make([]string, 0),
		Success:             false,
		Message:             "",
		CompletedSteps:      make([]pipeline.StepInfo, 0),
		Metadata:            make(map[string]string),
		TempFiles:           make([]string, 0),
		Logs:                make([]pipeline.LogEntry, 0),
	}
}

// buildExecutionMetadata builds execution metadata for the response
func (s *codeExecutionServiceImpl) buildExecutionMetadata(data *pipeline.ExecutionData) *pb.ExecutionMetadata {
	metadata := &pb.ExecutionMetadata{
		StartedAt:       timestamppb.New(data.StartTime),
		CompletedAt:     timestamppb.New(data.EndTime),
		ExecutionTimeMs: data.ExecutionTimeMS,
		MemoryUsedMb:    data.MemoryUsedMB,
		ExitCode:        int32(data.ExitCode),
	}

	// Add compilation info if available
	if data.CompilationResult != nil {
		metadata.Compilation = &pb.CompilationInfo{
			Success:           data.CompilationResult.Success,
			ErrorMessage:      data.CompilationResult.ErrorMessage,
			Warnings:          data.CompilationResult.Warnings,
			CompilationTimeMs: data.CompilationResult.CompilationTimeMS,
		}
	}

	// Add test results if available
	if data.TestResults != nil {
		testResults := make([]*pb.TestResult, len(data.TestResults))
		for i, result := range data.TestResults {
			testResults[i] = &pb.TestResult{
				TestId:          result.TestID,
				Passed:          result.Passed,
				ExpectedOutput:  result.ExpectedOutput,
				ActualOutput:    result.ActualOutput,
				ErrorMessage:    result.ErrorMessage,
				ExecutionTimeMs: result.ExecutionTimeMS,
			}
		}
		metadata.TestResults = testResults
	}

	return metadata
}

// buildPipelineSteps builds pipeline steps for the response
func (s *codeExecutionServiceImpl) buildPipelineSteps(data *pipeline.ExecutionData) []*pb.PipelineStep {
	steps := make([]*pb.PipelineStep, len(data.CompletedSteps))

	for i, stepInfo := range data.CompletedSteps {
		steps[i] = &pb.PipelineStep{
			Name:         stepInfo.Name,
			Status:       s.convertStepStatus(stepInfo.Status),
			StartedAt:    timestamppb.New(stepInfo.StartedAt),
			CompletedAt:  timestamppb.New(stepInfo.CompletedAt),
			Message:      stepInfo.Message,
			Error:        stepInfo.Error,
			StepMetadata: stepInfo.Metadata,
			StepOrder:    int32(stepInfo.Order),
		}
	}

	return steps
}

// convertExecutionStatus converts pipeline execution status to proto status
func (s *codeExecutionServiceImpl) convertExecutionStatus(status pipeline.ExecutionStatus) pb.ExecutionStatus {
	switch status {
	case pipeline.ExecutionStatusPending:
		return pb.ExecutionStatus_EXECUTION_STATUS_PENDING
	case pipeline.ExecutionStatusRunning:
		return pb.ExecutionStatus_EXECUTION_STATUS_RUNNING
	case pipeline.ExecutionStatusCompleted:
		return pb.ExecutionStatus_EXECUTION_STATUS_COMPLETED
	case pipeline.ExecutionStatusFailed:
		return pb.ExecutionStatus_EXECUTION_STATUS_FAILED
	case pipeline.ExecutionStatusTimeout:
		return pb.ExecutionStatus_EXECUTION_STATUS_TIMEOUT
	case pipeline.ExecutionStatusCancelled:
		return pb.ExecutionStatus_EXECUTION_STATUS_CANCELLED
	default:
		return pb.ExecutionStatus_EXECUTION_STATUS_UNSPECIFIED
	}
}

// convertStepStatus converts pipeline step status to proto status
func (s *codeExecutionServiceImpl) convertStepStatus(status pipeline.StepStatus) pb.StepStatus {
	switch status {
	case pipeline.StepStatusPending:
		return pb.StepStatus_STEP_STATUS_PENDING
	case pipeline.StepStatusRunning:
		return pb.StepStatus_STEP_STATUS_RUNNING
	case pipeline.StepStatusCompleted:
		return pb.StepStatus_STEP_STATUS_COMPLETED
	case pipeline.StepStatusFailed:
		return pb.StepStatus_STEP_STATUS_FAILED
	case pipeline.StepStatusSkipped:
		return pb.StepStatus_STEP_STATUS_SKIPPED
	default:
		return pb.StepStatus_STEP_STATUS_UNSPECIFIED
	}
}

// convertLogLevel converts pipeline log level to proto log level
func (s *codeExecutionServiceImpl) convertLogLevel(level pipeline.LogLevel) pb.LogLevel {
	switch level {
	case pipeline.LogLevelDebug:
		return pb.LogLevel_LOG_LEVEL_DEBUG
	case pipeline.LogLevelInfo:
		return pb.LogLevel_LOG_LEVEL_INFO
	case pipeline.LogLevelWarn:
		return pb.LogLevel_LOG_LEVEL_WARN
	case pipeline.LogLevelError:
		return pb.LogLevel_LOG_LEVEL_ERROR
	default:
		return pb.LogLevel_LOG_LEVEL_UNSPECIFIED
	}
}

// generateExecutionID generates a unique execution ID
func generateExecutionID() string {
	return fmt.Sprintf("exec_%d", time.Now().UnixNano())
}

// Server represents the gRPC server
type Server struct {
	grpcServer       *grpc.Server
	listener         net.Listener
	logger           pipeline.Logger
	database         *database.Database
	executionService *services.ExecutionService
}

// NewServer creates a new gRPC server
func NewServer(port string, challengesAPIURL string) (*Server, error) {
	// Create logger
	logger, err := pipeline.NewDefaultLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Log PORT environment variable for debugging
	fmt.Println("Environment PORT:", port)

	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create service implementation
	serviceServer := NewCodeExecutionServiceServer(logger, challengesAPIURL)

	// Register service using generated registration function
	pb.RegisterCodeExecutionServiceServer(grpcServer, serviceServer)

	// Log successful service registration
	logger.Info(context.Background(), "gRPC service registered successfully", map[string]interface{}{
		"service": "com.levelupjourney.coderunner.CodeExecutionService",
		"methods": []string{"ExecuteCode", "GetExecutionStatus", "HealthCheck", "StreamExecutionLogs"},
	})

	return &Server{
		grpcServer: grpcServer,
		listener:   lis,
		logger:     logger,
	}, nil
}

// NewServerWithDB creates a new gRPC server with database support
func NewServerWithDB(port string, challengesAPIURL string, db *database.Database) (*Server, error) {
	// Create logger
	logger, err := pipeline.NewDefaultLogger()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Create listener
	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %s: %w", port, err)
	}

	// Create repository and service
	executionRepo := repository.NewExecutionRepository(db.DB)
	executionService := services.NewExecutionServiceWithDB(executionRepo, db.DB)

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Create service implementation with database support
	serviceServer := NewCodeExecutionServiceServerWithDB(logger, challengesAPIURL, executionService)

	// Register service using generated registration function
	pb.RegisterCodeExecutionServiceServer(grpcServer, serviceServer)

	// Log successful service registration with database
	logger.Info(context.Background(), "gRPC service with database registered successfully", map[string]interface{}{
		"service":  "com.levelupjourney.coderunner.CodeExecutionService",
		"methods":  []string{"ExecuteCode", "GetExecutionStatus", "HealthCheck", "StreamExecutionLogs"},
		"database": "enabled",
	})

	return &Server{
		grpcServer:       grpcServer,
		listener:         lis,
		logger:           logger,
		database:         db,
		executionService: executionService,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	address := s.listener.Addr().String()

	s.logger.Info(context.Background(), "Starting gRPC server", map[string]interface{}{
		"address":  address,
		"protocol": "gRPC (HTTP/2)",
	})

	// Enhanced connection logs
	fmt.Printf("✅ gRPC Server Successfully Started!\n")
	fmt.Printf("🔗 Connection Details:\n")
	fmt.Printf("   Full URL: grpc://%s\n", address)
	fmt.Printf("   Protocol: gRPC over HTTP/2\n")
	fmt.Printf("   Security: Plaintext (no TLS)\n")
	fmt.Printf("   Status: SERVING\n")
	fmt.Printf("\n🎯 Ready to accept connections!\n")
	fmt.Printf("   Use any gRPC client to connect to: grpc://%s\n\n", address)

	// Start server in goroutine
	go func() {
		if err := s.grpcServer.Serve(s.listener); err != nil {
			log.Printf("❌ Failed to serve gRPC server: %v", err)
		}
	}()

	// Wait for interrupt signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	fmt.Printf("\n🛑 Shutdown signal received\n")
	s.logger.Info(context.Background(), "Shutting down gRPC server", map[string]interface{}{
		"address": address,
	})
	s.grpcServer.GracefulStop()

	return nil
}

// Stop stops the gRPC server
func (s *Server) Stop() {
	s.grpcServer.GracefulStop()

	// Close database connection if exists
	if s.database != nil {
		if err := s.database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}
}
