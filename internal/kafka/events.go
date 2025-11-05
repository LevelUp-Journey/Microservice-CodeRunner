package kafka

import (
	"context"
	"encoding/json"
	"time"
)

// ChallengeCompletedEvent represents an event when a challenge is completed
type ChallengeCompletedEvent struct {
	ChallengeID   string    `json:"challengeId"`
	UserID        string    `json:"userId"`
	ExecutionID   string    `json:"executionId"`
	Status        string    `json:"status"`
	Score         int       `json:"score"`
	TotalTests    int       `json:"totalTests"`
	PassedTests   int       `json:"passedTests"`
	ExecutionTime int64     `json:"executionTimeMs"`
	Timestamp     time.Time `json:"timestamp"`
}

// CodeExecutionEvent represents a general code execution event
type CodeExecutionEvent struct {
	ExecutionID string                 `json:"executionId"`
	Language    string                 `json:"language"`
	Status      string                 `json:"status"`
	Output      string                 `json:"output,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
}

// PublishChallengeCompleted publishes a challenge completed event to Kafka
// Uses the default topic from configuration
func (kc *KafkaClient) PublishChallengeCompleted(ctx context.Context, event *ChallengeCompletedEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessageToDefaultTopic(ctx, event.ChallengeID, data)
}

// PublishChallengeCompletedToTopic publishes a challenge completed event to a specific topic
func (kc *KafkaClient) PublishChallengeCompletedToTopic(ctx context.Context, topic string, event *ChallengeCompletedEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessage(ctx, topic, event.ChallengeID, data)
}

// PublishCodeExecution publishes a code execution event to Kafka
// Uses the default topic from configuration
func (kc *KafkaClient) PublishCodeExecution(ctx context.Context, event *CodeExecutionEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessageToDefaultTopic(ctx, event.ExecutionID, data)
}

// PublishCodeExecutionToTopic publishes a code execution event to a specific topic
func (kc *KafkaClient) PublishCodeExecutionToTopic(ctx context.Context, topic string, event *CodeExecutionEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessage(ctx, topic, event.ExecutionID, data)
}
