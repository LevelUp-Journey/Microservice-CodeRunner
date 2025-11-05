package env

import "time"

// Config holds all configuration for the application
type Config struct {
	App              AppConfig              `mapstructure:",squash"`
	Server           ServerConfig           `mapstructure:",squash"`
	Database         DatabaseConfig         `mapstructure:",squash"`
	Logging          LoggingConfig          `mapstructure:",squash"`
	Kafka            KafkaConfig            `mapstructure:",squash"`
	ServiceDiscovery ServiceDiscoveryConfig `mapstructure:",squash"`
}

// AppConfig holds application configuration
type AppConfig struct {
	Name    string `mapstructure:"APP_NAME"`
	Version string `mapstructure:"API_VERSION"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Port     string `mapstructure:"PORT"`
	GRPCPort string `mapstructure:"GRPC_PORT"`
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Host            string        `mapstructure:"DB_HOST"`
	Port            string        `mapstructure:"DB_PORT"`
	User            string        `mapstructure:"DB_USER"`
	Password        string        `mapstructure:"DB_PASSWORD"`
	Name            string        `mapstructure:"DB_NAME"`
	SSLMode         string        `mapstructure:"DB_SSLMODE"`
	Timezone        string        `mapstructure:"DB_TIMEZONE"`
	MaxOpenConns    int           `mapstructure:"DB_MAX_OPEN_CONNS"`
	MaxIdleConns    int           `mapstructure:"DB_MAX_IDLE_CONNS"`
	ConnMaxLifetime time.Duration `mapstructure:"DB_CONN_MAX_LIFETIME"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"LOG_LEVEL"`
	Format string `mapstructure:"LOG_FORMAT"`
}

// KafkaConfig holds Kafka/Event Hub configuration
type KafkaConfig struct {
	BootstrapServers  string `mapstructure:"KAFKA_BOOTSTRAP_SERVERS"`
	ConnectionString  string `mapstructure:"KAFKA_CONNECTION_STRING"`
	Topic             string `mapstructure:"KAFKA_TOPIC"`
	ConsumerGroup     string `mapstructure:"KAFKA_CONSUMER_GROUP"`
	SASLMechanism     string `mapstructure:"KAFKA_SASL_MECHANISM"`
	SecurityProtocol  string `mapstructure:"KAFKA_SECURITY_PROTOCOL"`
	ProducerTimeoutMs int    `mapstructure:"KAFKA_PRODUCER_TIMEOUT_MS"`
	ConsumerTimeoutMs int    `mapstructure:"KAFKA_CONSUMER_TIMEOUT_MS"`
	MaxRetries        int    `mapstructure:"KAFKA_MAX_RETRIES"`
}

// ServiceDiscoveryConfig holds service discovery configuration
type ServiceDiscoveryConfig struct {
	URL         string `mapstructure:"SERVICE_DISCOVERY_URL"`
	Enabled     bool   `mapstructure:"SERVICE_DISCOVERY_ENABLED"`
	PublicIP    string `mapstructure:"SERVICE_PUBLIC_IP"`
	ServiceName string `mapstructure:"SERVICE_NAME"`
	Hostname    string `mapstructure:"HOSTNAME"`
}
