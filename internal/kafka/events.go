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
func (kc *KafkaClient) PublishChallengeCompleted(ctx context.Context, event *ChallengeCompletedEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessage(ctx, event.ChallengeID, data)
}

// PublishCodeExecution publishes a code execution event to Kafka
func (kc *KafkaClient) PublishCodeExecution(ctx context.Context, event *CodeExecutionEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessage(ctx, event.ExecutionID, data)
}
