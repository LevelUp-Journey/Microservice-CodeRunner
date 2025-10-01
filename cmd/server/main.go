package main

import (
	"context"
	"log"
	"os"
	"time"

	"code-runner/env"
	"code-runner/internal/database"
	"code-runner/internal/docker"
	"code-runner/internal/server"
)

func main() {
	// Load configuration
	config, err := env.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	if err := database.InitDB(&config.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := database.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Initialize Docker executor and verify images
	log.Printf("🐳 Initializing Docker environment...")
	dockerExecutor, err := docker.NewDockerExecutor()
	if err != nil {
		log.Fatalf("Failed to create Docker executor: %v", err)
	}

	// Verify and build Docker images if necessary
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	if err := dockerExecutor.EnsureImagesReady(ctx); err != nil {
		log.Fatalf("Failed to ensure Docker images are ready: %v", err)
	}

	// Print startup information
	log.Printf("🚀 Starting %s gRPC Server", config.App.Name)
	log.Printf("📍 Port: %s", config.Server.GRPCPort)
	log.Printf("🔧 Configuration: plaintext negotiation, 8MB max message size")
	log.Printf("🌐 Client connection: static://localhost:%s", config.Server.GRPCPort)
	log.Printf("🗄️  Database: %s:%s/%s", config.Database.Host, config.Database.Port, config.Database.Name)

	// Override with environment variable if set (for backward compatibility)
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = config.Server.GRPCPort
	}

	// Start gRPC server
	if err := server.StartServer(port, database.GetDB()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
