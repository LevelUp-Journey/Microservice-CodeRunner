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
	// Database configuration
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBTimeZone string
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
		// Database configuration
		DBHost:     getEnv("DB_HOST", "localhost"),
		DBPort:     getEnv("DB_PORT", "5432"),
		DBUser:     getEnv("DB_USER", "postgres"),
		DBPassword: getEnv("DB_PASSWORD", "postgres"),
		DBName:     getEnv("DB_NAME", "coderunner"),
		DBSSLMode:  getEnv("DB_SSLMODE", "disable"),
		DBTimeZone: getEnv("DB_TIMEZONE", "UTC"),
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

// GetDatabaseDSN returns the database connection string
func GetDatabaseDSN() string {
	if AppConfig == nil {
		return "host=localhost user=postgres password=postgres dbname=coderunner port=5432 sslmode=disable TimeZone=UTC"
	}
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=%s",
		AppConfig.DBHost,
		AppConfig.DBUser,
		AppConfig.DBPassword,
		AppConfig.DBName,
		AppConfig.DBPort,
		AppConfig.DBSSLMode,
		AppConfig.DBTimeZone,
	)
}
