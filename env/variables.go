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
	APIVersion string
	BasePath   string // Full URL for external references
	APIPath    string // Just the path for routing
	Host       string
	AppName    string
}

var AppConfig *Config

func LoadConfig() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Println("⚠️ There was an error loading .env")
	}

	AppConfig = &Config{
		Port:       getEnv("PORT", "8084"),
		APIVersion: getEnv("API_VERSION", "v1"),
		BasePath:   getEnv("APP_BASEPATH", "/api/v1"),
		Host:       getEnv("APP_HOST", "localhost"),
		AppName:    getEnv("APP_NAME", "Microservice CodeRunner API"),
	}

	// Pretty-print config as JSON
	printConfig(AppConfig)
}

// Helper
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}

func printConfig(c *Config) {
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		fmt.Println("❌ Error marshalling config:", err)
		return
	}
	fmt.Println("✅ AppConfig Loaded:")
	fmt.Println(string(b))
}
