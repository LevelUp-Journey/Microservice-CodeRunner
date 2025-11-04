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
			Name:    getEnv("APP_NAME", "microservice-code-runner"),
			Version: getEnv("API_VERSION", "v1"),
		},
		Server: ServerConfig{
			Port:     getEnv("PORT", "8084"),
			GRPCPort: getEnv("GRPC_PORT", "9084"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "postgres"),
			Name:     getEnv("DB_NAME", "code_runner_db"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
			Timezone: getEnv("DB_TIMEZONE", "UTC"),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Kafka: KafkaConfig{
			BootstrapServers:  getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
			ConnectionString:  getEnv("KAFKA_CONNECTION_STRING", ""),
			Topic:             getEnv("KAFKA_TOPIC", "challenge.completed"),
			ConsumerGroup:     getEnv("KAFKA_CONSUMER_GROUP", "code-runner-service"),
			SASLMechanism:     getEnv("KAFKA_SASL_MECHANISM", "PLAIN"),
			SecurityProtocol:  getEnv("KAFKA_SECURITY_PROTOCOL", "SASL_SSL"),
			ProducerTimeoutMs: getEnvInt("KAFKA_PRODUCER_TIMEOUT_MS", 30000),
			ConsumerTimeoutMs: getEnvInt("KAFKA_CONSUMER_TIMEOUT_MS", 30000),
			MaxRetries:        getEnvInt("KAFKA_MAX_RETRIES", 3),
		},
		ServiceDiscovery: ServiceDiscoveryConfig{
			URL:      getEnv("SERVICE_DISCOVERY_URL", ""),
			Enabled:  getEnvBool("SERVICE_DISCOVERY_ENABLED", false),
			PublicIP: getEnv("SERVICE_PUBLIC_IP", ""),
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

// getEnvInt gets an environment variable as int with a fallback value
func getEnvInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if val, err := strconv.Atoi(value); err == nil {
			return val
		}
	}
	return fallback
}

// getEnvBool gets an environment variable as bool with a fallback value
func getEnvBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if val, err := strconv.ParseBool(value); err == nil {
			return val
		}
	}
	return fallback
}
