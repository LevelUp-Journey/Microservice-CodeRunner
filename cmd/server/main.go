package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
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

	// 🚀 Registrar en Eureka vía API REST
	eurekaURL := os.Getenv("SERVICE_DISCOVERY_URL")
	if eurekaURL == "" {
		eurekaURL = "http://localhost:8761/eureka"
	} else {
		eurekaURL = strings.TrimRight(eurekaURL, "/") // ✅ elimina barras finales
	}

	ip, err := getLocalIP()
	if err != nil {
		log.Fatalf("Failed to get local IP: %v", err)
	}

	port := os.Getenv("GRPC_PORT")
	if port == "" {
		port = config.Server.GRPCPort
	}

	portInt, err := strconv.Atoi(port)
	if err != nil {
		log.Fatalf("Invalid GRPC_PORT: %v", err)
	}

	// Estructuras para el JSON de registro
	type DataCenterInfo struct {
		Class string `json:"@class"`
		Name  string `json:"name"`
	}

	type PortInfo struct {
		Port    int  `json:"$"`
		Enabled bool `json:"@enabled"`
	}

	type Instance struct {
		HostName       string         `json:"hostName"`
		App            string         `json:"app"`
		IPAddr         string         `json:"ipAddr"`
		VipAddress     string         `json:"vipAddress"`
		Status         string         `json:"status"`
		Port           PortInfo       `json:"port"`
		DataCenterInfo DataCenterInfo `json:"dataCenterInfo"`
	}

	type EurekaRequest struct {
		Instance Instance `json:"instance"`
	}

	// Crear la instancia
	instanceData := EurekaRequest{
		Instance: Instance{
			HostName:   ip, // Usar IP como hostname
			App:        "CODE_RUNNER_SERVICE",
			IPAddr:     ip,
			VipAddress: "CODE_RUNNER_SERVICE",
			Status:     "UP",
			Port:       PortInfo{Port: portInt, Enabled: true},
			DataCenterInfo: DataCenterInfo{
				Class: "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
				Name:  "MyOwn",
			},
		},
	}

	// Serializar a JSON
	jsonData, err := json.Marshal(instanceData)
	if err != nil {
		log.Fatalf("Failed to marshal instance data: %v", err)
	}

	// Registrar vía POST
	registerURL := eurekaURL + "/apps/" + instanceData.Instance.App
	log.Printf("🔍 Attempting to register at: %s", registerURL)
	resp, err := http.Post(registerURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Error registering with Eureka: %v", err)
	} else {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
			log.Printf("✅ Service registered in Eureka as CODE_RUNNER_SERVICE at %s:%d", ip, portInt)
		} else {
			log.Printf("❌ Registration failed with status: %d", resp.StatusCode)
		}
	}

	// Iniciar goroutine para heartbeats
	go func() {
		heartbeatURL := registerURL + "/" + instanceData.Instance.HostName
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			req, err := http.NewRequest("PUT", heartbeatURL, nil)
			if err != nil {
				log.Printf("❌ Failed to create heartbeat request: %v", err)
				continue
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("❌ Heartbeat failed: %v", err)
			} else {
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
					log.Printf("❌ Heartbeat failed with status: %d", resp.StatusCode)
				}
			}
		}
	}()

	// Print startup information
	log.Printf("🚀 Starting %s gRPC Server", config.App.Name)
	log.Printf("📍 Port: %s", config.Server.GRPCPort)
	log.Printf("🔧 Configuration: plaintext negotiation, 8MB max message size")
	log.Printf("🌐 Client connection: static://localhost:%s", config.Server.GRPCPort)
	log.Printf("🗄️  Database: %s:%s/%s", config.Database.Host, config.Database.Port, config.Database.Name)
	log.Printf("🔍 Eureka: Registered at %s", eurekaURL)

	// Start gRPC server
	if err := server.StartServer(port, database.GetDB()); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}

// Función auxiliar para obtener la IP local
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}
