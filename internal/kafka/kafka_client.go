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
	readers  map[string]*kafka.Reader // Support multiple topics
	producer *kafka.Writer
	dialer   *kafka.Dialer
}

// NewKafkaClient creates a new Kafka client configured for Azure Event Hub
func NewKafkaClient(config *env.KafkaConfig) (*KafkaClient, error) {
	if config.BootstrapServers == "" {
		return nil, fmt.Errorf("KAFKA_BOOTSTRAP_SERVERS is required")
	}

	client := &KafkaClient{
		config:  config,
		readers: make(map[string]*kafka.Reader),
	}

	// Initialize dialer
	if err := client.initDialer(); err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka dialer: %w", err)
	}

	// Initialize producer (no specific topic needed for producer)
	if err := client.initProducer(); err != nil {
		return nil, fmt.Errorf("failed to initialize Kafka producer: %w", err)
	}

	log.Printf("✅ Kafka client initialized successfully")
	log.Printf("📡 Bootstrap servers: %s", config.BootstrapServers)
	log.Printf("🔧 Ready for dynamic topic operations")

	return client, nil
}

// initDialer initializes the Kafka dialer with SASL/SSL configuration
func (kc *KafkaClient) initDialer() error {
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
	kc.dialer = &kafka.Dialer{
		Timeout:       10 * time.Second,
		DualStack:     true,
		SASLMechanism: mechanism,
		TLS:           &tls.Config{},
	}

	return nil
}

// initProducer initializes the Kafka producer without a specific topic
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

	// Create writer (producer) without topic (will specify per message)
	kc.writer = &kafka.Writer{
		Addr:         kafka.TCP(kc.config.BootstrapServers),
		Balancer:     &kafka.LeastBytes{},
		WriteTimeout: time.Duration(kc.config.ProducerTimeoutMs) * time.Millisecond,
		ReadTimeout:  time.Duration(kc.config.ProducerTimeoutMs) * time.Millisecond,
		Transport: &kafka.Transport{
			SASL: mechanism,
			TLS:  &tls.Config{},
			Dial: kc.dialer.DialFunc,
		},
		MaxAttempts:            kc.config.MaxRetries,
		RequiredAcks:           kafka.RequireAll,
		Compression:            kafka.Snappy,
		AllowAutoTopicCreation: true, // Allow automatic topic creation
	}

	kc.producer = kc.writer

	return nil
}

// InitConsumer initializes a Kafka consumer for a specific topic
func (kc *KafkaClient) InitConsumer(topic string) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	// Check if consumer for this topic already exists
	if _, exists := kc.readers[topic]; exists {
		log.Printf("ℹ️  Consumer for topic '%s' already exists", topic)
		return nil
	}

	// Create reader (consumer) for the specific topic
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        []string{kc.config.BootstrapServers},
		Topic:          topic,
		GroupID:        kc.config.ConsumerGroup,
		MinBytes:       10e3, // 10KB
		MaxBytes:       10e6, // 10MB
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
		Dialer:         kc.dialer,
	})

	kc.readers[topic] = reader

	log.Printf("✅ Kafka consumer initialized for topic: %s", topic)
	log.Printf("👥 Consumer group: %s", kc.config.ConsumerGroup)

	return nil
}

// InitConsumerForTopics initializes consumers for multiple topics
func (kc *KafkaClient) InitConsumerForTopics(topics []string) error {
	for _, topic := range topics {
		if err := kc.InitConsumer(topic); err != nil {
			return fmt.Errorf("failed to initialize consumer for topic %s: %w", topic, err)
		}
	}
	return nil
}

// ProduceMessage sends a message to a specific Kafka topic
func (kc *KafkaClient) ProduceMessage(ctx context.Context, topic string, key string, value []byte) error {
	if kc.writer == nil {
		return fmt.Errorf("kafka producer not initialized")
	}

	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}

	message := kafka.Message{
		Topic: topic,
		Key:   []byte(key),
		Value: value,
		Time:  time.Now(),
	}

	err := kc.writer.WriteMessages(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to write message to topic %s: %w", topic, err)
	}

	log.Printf("✉️  Message sent to topic '%s' - Key: %s, Size: %d bytes", topic, key, len(value))
	return nil
}

// ProduceMessageToDefaultTopic sends a message to the default topic from config
func (kc *KafkaClient) ProduceMessageToDefaultTopic(ctx context.Context, key string, value []byte) error {
	return kc.ProduceMessage(ctx, kc.config.Topic, key, value)
}

// ConsumeMessages starts consuming messages from a specific topic
func (kc *KafkaClient) ConsumeMessages(ctx context.Context, topic string, handler func(kafka.Message) error) error {
	reader, exists := kc.readers[topic]
	if !exists {
		return fmt.Errorf("consumer for topic '%s' not initialized. Call InitConsumer first", topic)
	}

	log.Printf("🎧 Starting Kafka consumer for topic: %s", topic)

	for {
		select {
		case <-ctx.Done():
			log.Printf("🛑 Stopping Kafka consumer for topic: %s", topic)
			return ctx.Err()
		default:
			message, err := reader.ReadMessage(ctx)
			if err != nil {
				log.Printf("❌ Error reading message from topic %s: %v", topic, err)
				continue
			}

			log.Printf("📨 Received message - Topic: %s, Partition: %d, Offset: %d",
				message.Topic, message.Partition, message.Offset)

			if err := handler(message); err != nil {
				log.Printf("❌ Error handling message from topic %s: %v", topic, err)
				// Continue processing other messages
			}
		}
	}
}

// ConsumeFromMultipleTopics starts consuming from multiple topics concurrently
func (kc *KafkaClient) ConsumeFromMultipleTopics(ctx context.Context, topics []string, handler func(kafka.Message) error) error {
	if len(topics) == 0 {
		return fmt.Errorf("no topics provided")
	}

	// Start a goroutine for each topic
	errChan := make(chan error, len(topics))

	for _, topic := range topics {
		go func(t string) {
			if err := kc.ConsumeMessages(ctx, t, handler); err != nil {
				errChan <- fmt.Errorf("error consuming from topic %s: %w", t, err)
			}
		}(topic)
	}

	// Wait for context cancellation or first error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errChan:
		return err
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

	// Close all readers
	for topic, reader := range kc.readers {
		if err := reader.Close(); err != nil {
			errs = append(errs, fmt.Errorf("failed to close reader for topic %s: %w", topic, err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing Kafka client: %v", errs)
	}

	log.Printf("✅ Kafka client closed successfully")
	return nil
}

// parseConnectionString parses Azure Event Hub connection string
// Format: Endpoint=sb://...;SharedAccessKeyName=...;SharedAccessKey=...
// Note: EntityPath is optional and not required in connection string
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
		return "", "", fmt.Errorf("invalid connection string format: missing SharedAccessKeyName or SharedAccessKey")
	}

	// For Azure Event Hub, username is "$ConnectionString" and password is the full connection string
	username = "$ConnectionString"
	password = connStr

	return username, password, nil
}
