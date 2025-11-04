package kafka

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"strings"
	"time"

	"code-runner/env"

	"github.com/segmentio/kafka-go"
	"github.com/segmentio/kafka-go/sasl/plain"
)

// KafkaClient wraps Kafka producer and consumer functionality
type KafkaClient struct {
	config   *env.KafkaConfig
	writer   *kafka.Writer
	reader   *kafka.Reader
	producer *kafka.Writer
}

// NewKafkaClient creates a new Kafka client configured for Azure Event Hub
func NewKafkaClient(config *env.KafkaConfig) (*KafkaClient, error) {
	if config.BootstrapServers == "" {
		return nil, fmt.Errorf("KAFKA_BOOTSTRAP_SERVERS is required")
	}

	client := &KafkaClient{
		config: config,
	}

	// Initialize producer
	if err := client.initProducer(); err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}

	log.Printf("✅ Kafka client initialized successfully")
	log.Printf("📡 Bootstrap servers: %s", config.BootstrapServers)
	log.Printf("📝 Topic: %s", config.Topic)

	return client, nil
}

// initProducer initializes the Kafka producer
func (kc *KafkaClient) initProducer() error {
	// Parse connection string to get credentials
	username, password, err := parseConnectionString(kc.config.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Create SASL mechanism
	mechanism := plain.Mechanism{
		Username: username,
		Password: password,
	}

	// Configure TLS
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	// Create writer (producer)
	kc.writer = &kafka.Writer{
		Addr:         kafka.TCP(kc.config.BootstrapServers),
		Topic:        kc.config.Topic,
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: time.Duration(kc.config.ProducerTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(kc.config.ProducerTimeoutMs) * time.Millisecond,
		Transport: &kafka.Transport{
			SASL: mechanism,
			TLS:  &tls.Config{},
			Dial: dialer.DialFunc,
		},
		MaxAttempts:  kc.config.MaxRetries,
		RequiredAcks: kafka.RequireAll,
		Compression:  kafka.Snappy,
	}

	kc.producer = kc.writer

	return nil
}

// InitConsumer initializes a Kafka consumer
func (kc *KafkaClient) InitConsumer() error {
	// Parse connection string to get credentials
	username, password, err := parseConnectionString(kc.config.ConnectionString)
	if err != nil {
		return fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Create SASL mechanism
	mechanism := plain.Mechanism{
		Username: username,
		Password: password,
	}

	// Configure TLS
	dialer := &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	// Create reader (consumer)
	kc.reader = kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kc.config.BootstrapServers},
		Topic:          kc.config.Topic,
		GroupID:        kc.config.ConsumerGroup,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		Dialer:         dialer,
	})

	log.Printf("✅ Kafka consumer initialized successfully")
	log.Printf("👥 Consumer group: %s", kc.config.ConsumerGroup)

	return nil
}

// ProduceMessage sends a message to Kafka
func (kc *KafkaClient) ProduceMessage(ctx context.Context, key string, value []byte) error {
	if kc.writer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	message := kafka.Message{
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	err := kc.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	log.Printf("✉️  Message sent to Kafka - Key: %s, Size: %d bytes", key, len(value))
	return nil
}

// ConsumeMessages starts consuming messages from Kafka
func (kc *KafkaClient) ConsumeMessages(ctx context.Context, handler func(kafka.Message) error) error {
	if kc.reader == nil {
		return fmt.Errorf("kafka consumer not initialized")
	}

	log.Printf("🎧 Starting Kafka consumer...")

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Stopping Kafka consumer...")
			return ctx.Err()
		default:
			message, err := kc.reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("❌ Error reading message: %v", err)
				continue
			}

			log.Printf("📨 Received message - Topic: %s, Partition: %d, Offset: %d",
				message.Topic, message.Partition, message.Offset)

			if err := handler(message); err != nil {
				log.Printf("❌ Error handling message: %v", err)
				// Continue processing other messages
			}
		}
	}
}

// Close closes all Kafka connections
func (kc *KafkaClient) Close() error {
	var errs []error

	if kc.writer != nil {
		if err := kc.writer.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close writer: %w", err))
		}
	}

	if kc.reader != nil {
		if err := kc.reader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close reader: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing Kafka client: %v", errs)
	}

	log.Printf("✅ Kafka client closed successfully")
	return nil
}

// parseConnectionString parses Azure Event Hub connection string
// Format: Endpoint=sb://...;SharedAccessKeyName=...;SharedAccessKey=...;EntityPath=...
func parseConnectionString(connStr string) (username, password string, err error) {
	if connStr == "" {
		return "", "", fmt.Errorf("connection string is empty")
	}

	parts := strings.Split(connStr, ";")
	var keyName, keyValue string

	for _, part := range parts {
		if strings.HasPrefix(part, "SharedAccessKeyName=") {
			keyName = strings.TrimPrefix(part, "SharedAccessKeyName=")
		}
		if strings.HasPrefix(part, "SharedAccessKey=") {
			keyValue = strings.TrimPrefix(part, "SharedAccessKey=")
		}
	}

	if keyName == "" || keyValue == "" {
		return "", "", fmt.Errorf("invalid connection string format")
	}

	// For Azure Event Hub, username is "$ConnectionString" and password is the full connection string
	// Alternative: use SharedAccessKeyName as username and SharedAccessKey as password
	username = "$ConnectionString"
	password = connStr

	return username, password, nil
}
