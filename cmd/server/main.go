package main

import (
	"log"
	"os"

	"code-runner/internal/server"
)

func main() {
	// Get port from environment or use default
	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = "50051"
	}

	// Print startup information
	log.Println("🚀 Starting simplified Code Runner gRPC Server (adapter only)")
	log.Printf("📍 Port: %s", port)
	log.Printf("🌐 Full URL: grpc://localhost:%s", port)

	// Start simplified gRPC server
	if err := server.StartServer(port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
