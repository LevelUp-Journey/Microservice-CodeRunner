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

// ExecutionMetricsEvent representa las métricas detalladas de una ejecución de código
type ExecutionMetricsEvent struct {
	// Identificadores
	ExecutionID   string `json:"execution_id"`
	ChallengeID   string `json:"challenge_id"`
	CodeVersionID string `json:"code_version_id"`
	StudentID     string `json:"student_id"`

	// Información de la ejecución
	Language  string    `json:"language"`
	Status    string    `json:"status"` // pending, running, completed, failed, timed_out, cancelled
	Timestamp time.Time `json:"timestamp"`

	// Métricas de rendimiento
	ExecutionTimeMS int64   `json:"execution_time_ms"`
	MemoryUsageMB   float64 `json:"memory_usage_mb,omitempty"`
	ExitCode        int     `json:"exit_code,omitempty"`

	// Resultados de tests
	TotalTests  int  `json:"total_tests"`
	PassedTests int  `json:"passed_tests"`
	FailedTests int  `json:"failed_tests"`
	Success     bool `json:"success"`

	// Información de errores
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorType    string `json:"error_type,omitempty"` // compilation_error, runtime_error, timeout, etc.

	// Pasos de compilación (si aplica)
	CompilationSuccess  bool   `json:"compilation_success,omitempty"`
	CompilationTimeMS   int64  `json:"compilation_time_ms,omitempty"`
	CompilationError    string `json:"compilation_error,omitempty"`
	CompilationWarnings int    `json:"compilation_warnings,omitempty"`

	// Detalles de tests individuales
	TestResults []TestResultMetric `json:"test_results,omitempty"`

	// Metadata adicional
	ServerInstance string            `json:"server_instance,omitempty"`
	ClientIP       string            `json:"client_ip,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// TestResultMetric representa las métricas de un test individual
type TestResultMetric struct {
	TestID          string `json:"test_id"`
	TestName        string `json:"test_name,omitempty"`
	Passed          bool   `json:"passed"`
	ExecutionTimeMS int64  `json:"execution_time_ms,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// PublishExecutionMetrics publica métricas detalladas de ejecución a Kafka
func (kc *KafkaClient) PublishExecutionMetrics(ctx context.Context, event *ExecutionMetricsEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Usar el execution_id como key para mantener el orden de eventos del mismo execution
	return kc.ProduceMessageToDefaultTopic(ctx, event.ExecutionID, data)
}

// PublishExecutionMetricsToTopic publica métricas a un topic específico
func (kc *KafkaClient) PublishExecutionMetricsToTopic(ctx context.Context, topic string, event *ExecutionMetricsEvent) error {
	event.Timestamp = time.Now()

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}

	return kc.ProduceMessage(ctx, topic, event.ExecutionID, data)
}
