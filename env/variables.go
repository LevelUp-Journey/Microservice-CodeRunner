package variables

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port       string
	APIVersion string
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
		APIVersion: getEnv("API_VERSION", "1"),
	}
}

// Helper
func getEnv(key, defaultVal string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultVal
}
