package env

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: .env file not found or could not be loaded: %v", err)
	}

	config := &Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "microservice-challenges"),
			Version: getEnv("API_VERSION", "v1"),
		},
		Server: ServerConfig{
			Port:     getEnv("PORT", "8084"),
			GRPCPort: getEnv("GRPC_PORT", "9084"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "admin"),
			Password: getEnv("DB_PASSWORD", "admin"),
			Name:     getEnv("DB_NAME", "challenges_microservice"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			Timezone: getEnv("DB_TIMEZONE", "UTC"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
	}

	// Parse database connection pool settings
	if maxOpenConns := getEnv("DB_MAX_OPEN_CONNS", "25"); maxOpenConns != "" {
		if val, err := strconv.Atoi(maxOpenConns); err == nil {
			config.Database.MaxOpenConns = val
		}
	}

	if maxIdleConns := getEnv("DB_MAX_IDLE_CONNS", "10"); maxIdleConns != "" {
		if val, err := strconv.Atoi(maxIdleConns); err == nil {
			config.Database.MaxIdleConns = val
		}
	}

	if connMaxLifetime := getEnv("DB_CONN_MAX_LIFETIME", "3600"); connMaxLifetime != "" {
		if val, err := strconv.Atoi(connMaxLifetime); err == nil {
			config.Database.ConnMaxLifetime = time.Duration(val) * time.Second
		}
	}

	return config, nil
}

// getEnv gets an environment variable with a fallback value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
