package variables

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	APIVersion string
	BasePath   string
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
}

// Helper
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
