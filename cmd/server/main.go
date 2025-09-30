package main

import (
	"fmt"
	"log"

	variables "code-runner/env"
	"code-runner/internal/database"
	"code-runner/internal/server"
)

func main() {
	// Load configuration
	variables.LoadConfig()

	// Initialize database
	fmt.Println("🗄️  Initializing database...")
	db, err := database.Initialize()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Get port from configuration
	port := variables.GetGRPCPort()

	// Print startup information
	fmt.Println("🚀 Starting Code Runner gRPC Server")
	fmt.Printf("📍 Port: %s\n", port)
	fmt.Printf("📱 App: %s\n", variables.AppConfig.AppName)
	fmt.Printf("🗄️  Database: %s@%s:%s/%s\n",
		variables.AppConfig.DBUser,
		variables.AppConfig.DBHost,
		variables.AppConfig.DBPort,
		variables.AppConfig.DBName)

	// Create and start gRPC server with database
	srv, err := server.NewServerWithDB(port, "", db)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Print connection information before starting
	fmt.Printf("\n🌐 gRPC Server Connection Information:\n")
	fmt.Printf("   Protocol: gRPC (HTTP/2)\n")
	fmt.Printf("   Host: localhost\n")
	fmt.Printf("   Port: %s\n", port)
	fmt.Printf("   Full URL: grpc://localhost:%s\n", port)

	// Start server (this will block until interrupted)
	if err := srv.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
