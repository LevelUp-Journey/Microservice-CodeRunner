package variables

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	GRPCPort   string
	APIVersion string
	AppName    string
}

var AppConfig *Config

func LoadConfig() {
	// Load .env file if it exists
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️  No .env file found, using environment variables or defaults")
	}

	AppConfig = &Config{
		Port:       getEnv("PORT", "8084"),
		GRPCPort:   getEnv("GRPC_PORT", "9084"),
		APIVersion: getEnv("API_VERSION", "v1"),
		AppName:    getEnv("APP_NAME", "Microservice CodeRunner gRPC API"),
	}

	// Print config
	printConfig(AppConfig)
}

// getEnv gets environment variable or returns default
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

// printConfig prints the configuration in a readable format
func printConfig(c *Config) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("❌ Error marshalling config:", err)
		return
	}

	fmt.Println("✅ AppConfig Loaded:")
	fmt.Println(string(b))
}

// GetGRPCPort returns the gRPC port
func GetGRPCPort() string {
	if AppConfig == nil {
		return "9084"
	}
	return AppConfig.GRPCPort
}
