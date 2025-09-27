package main

import (
	"fmt"
	"log"

	variables "code-runner/env"
	"code-runner/internal/server"
)

func main() {
	// Load configuration
	variables.LoadConfig()

	// Get port from configuration
	port := variables.GetGRPCPort()

	// Print startup information
	fmt.Println("🚀 Starting Code Runner gRPC Server")
	fmt.Printf("Port: %s\n", port)
	fmt.Printf("App: %s\n", variables.AppConfig.AppName)

	// Create and start gRPC server
	srv, err := server.NewServer(port, "")
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Start server (this will block until interrupted)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
