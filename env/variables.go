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
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", "postgres"),
			Password:        getEnv("DB_PASSWORD", "postgres"),
			Name:            getEnv("DB_NAME", "code_runner_db"),
			SSLMode:         getEnv("DB_SSLMODE", "require"),
			Timezone:        getEnv("DB_TIMEZONE", "UTC"),
			MaxOpenConns:    25,
			MaxIdleConns:    10,
			ConnMaxLifetime: 3600 * time.Second,
		},
		Logging: LoggingConfig{
			Level:  "info",
			Format: "json",
		},
		Kafka: KafkaConfig{
			BootstrapServers:  getEnv("KAFKA_BOOTSTRAP_SERVERS", "localhost:9092"),
			ConnectionString:  getEnv("KAFKA_CONNECTION_STRING", ""),
			UseSASL:           getEnvBool("KAFKA_USE_SASL", false),
			Topic:             "challenge.completed",
			ConsumerGroup:     "code-runner-service",
			SASLMechanism:     "PLAIN",
			SecurityProtocol:  "SASL_SSL",
			ProducerTimeoutMs: 30000,
			ConsumerTimeoutMs: 30000,
			MaxRetries:        3,
		},
		ServiceDiscovery: ServiceDiscoveryConfig{
			URL:         getEnv("SERVICE_DISCOVERY_URL", ""),
			Enabled:     false, // Will be set to true if URL is provided
			PublicIP:    getEnv("SERVICE_PUBLIC_IP", ""),
			ServiceName: getEnv("SERVICE_NAME", "CODE-RUNNER-SERVICE"),
		},
	}

	// Auto-enable service discovery if URL is provided
	if config.ServiceDiscovery.URL != "" {
		config.ServiceDiscovery.Enabled = true
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
