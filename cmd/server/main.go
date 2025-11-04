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
"code-runner/internal/kafka"
"code-runner/internal/server"
)

func main() {
config, err := env.LoadConfig()
if err != nil {
log.Fatalf("Failed to load configuration: %v", err)
}

if err := database.InitDB(&config.Database); err != nil {
log.Fatalf("Failed to initialize database: %v", err)
}
defer func() {
if err := database.Close(); err != nil {
log.Printf("Error closing database: %v", err)
}
}()

log.Printf("📡 Initializing Kafka client...")
kafkaClient, err := kafka.NewKafkaClient(&config.Kafka)
if err != nil {
log.Printf("⚠️  Warning: Failed to initialize Kafka client: %v", err)
log.Printf("ℹ️  Continuing without Kafka support...")
} else {
defer func() {
if err := kafkaClient.Close(); err != nil {
log.Printf("Error closing Kafka client: %v", err)
}
}()
}

log.Printf("🐳 Initializing Docker environment...")
dockerExecutor, err := docker.NewDockerExecutor()
if err != nil {
log.Fatalf("Failed to create Docker executor: %v", err)
}

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

if err := dockerExecutor.EnsureImagesReady(ctx); err != nil {
log.Fatalf("Failed to ensure Docker images are ready: %v", err)
}

grpcPort := os.Getenv("GRPC_PORT")
if grpcPort == "" {
grpcPort = config.Server.GRPCPort
}

portInt, err := strconv.Atoi(grpcPort)
if err != nil {
log.Fatalf("Invalid GRPC_PORT: %v", err)
}

if config.ServiceDiscovery.Enabled && config.ServiceDiscovery.URL != "" {
eurekaURL := strings.TrimRight(config.ServiceDiscovery.URL, "/")
ip, err := getLocalIP()
if err != nil {
log.Printf("⚠️  Warning: Failed to get local IP: %v", err)
ip = "localhost"
}
go registerWithEureka(eurekaURL, ip, portInt)
} else {
log.Printf("ℹ️  Service Discovery is disabled")
}

log.Printf("🚀 Starting %s gRPC Server", config.App.Name)
log.Printf("📍 Port: %s", grpcPort)
log.Printf("🔧 Configuration: plaintext negotiation, 8MB max message size")
log.Printf("🌐 Client connection: static://localhost:%s", grpcPort)
log.Printf("🗄️  Database: %s:%s/%s", config.Database.Host, config.Database.Port, config.Database.Name)

if config.Kafka.BootstrapServers != "" {
log.Printf("📨 Kafka: %s", config.Kafka.BootstrapServers)
log.Printf("📝 Topic: %s", config.Kafka.Topic)
log.Printf("👥 Consumer Group: %s", config.Kafka.ConsumerGroup)
}

if config.ServiceDiscovery.Enabled && config.ServiceDiscovery.URL != "" {
log.Printf("🔍 Service Discovery: %s", config.ServiceDiscovery.URL)
}

if err := server.StartServer(grpcPort, database.GetDB()); err != nil {
log.Fatalf("Failed to start server: %v", err)
}

log.Println("Server stopped")
}

func registerWithEureka(eurekaURL, ip string, port int) {
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

instanceData := EurekaRequest{
Instance: Instance{
HostName:   ip,
App:        "CODE-RUNNER-SERVICE",
IPAddr:     ip,
VipAddress: "CODE-RUNNER-SERVICE",
Status:     "UP",
Port:       PortInfo{Port: port, Enabled: true},
DataCenterInfo: DataCenterInfo{
Class: "com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo",
Name:  "MyOwn",
},
},
}

jsonData, err := json.Marshal(instanceData)
if err != nil {
log.Printf("❌ Failed to marshal instance data: %v", err)
return
}

registerURL := eurekaURL + "/apps/" + instanceData.Instance.App
log.Printf("🔍 Attempting to register at: %s", registerURL)

resp, err := http.Post(registerURL, "application/json", bytes.NewBuffer(jsonData))
if err != nil {
log.Printf("❌ Error registering with Eureka: %v", err)
return
}
defer resp.Body.Close()

if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
log.Printf("✅ Service registered in Eureka as CODE-RUNNER-SERVICE at %s:%d", ip, port)
} else {
log.Printf("❌ Registration failed with status: %d", resp.StatusCode)
return
}

heartbeatURL := registerURL + "/" + instanceData.Instance.HostName
ticker := time.NewTicker(30 * time.Second)
defer ticker.Stop()

for range ticker.C {
req, err := http.NewRequest("PUT", heartbeatURL, nil)
if err != nil {
log.Printf("❌ Failed to create heartbeat request: %v", err)
continue
}

client := &http.Client{Timeout: 10 * time.Second}
resp, err := client.Do(req)
if err != nil {
log.Printf("❌ Heartbeat failed: %v", err)
continue
}
resp.Body.Close()

if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
log.Printf("💓 Heartbeat sent successfully to Eureka")
} else {
log.Printf("❌ Heartbeat failed with status: %d", resp.StatusCode)
}
}
}

func getLocalIP() (string, error) {
conn, err := net.Dial("udp", "8.8.8.8:80")
if err != nil {
return "", err
}
defer conn.Close()
localAddr := conn.LocalAddr().(*net.UDPAddr)
return localAddr.IP.String(), nil
}
