package analyzer

import (
	"context"
	"time"
)

// Provider is the interface that all LLM providers implement.
type Provider interface {
	Name() string
	Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error)
	EstimateCost(inputTokens, outputTokens int, modelID string) float64
}

// CompletionRequest contains parameters for a single LLM call.
type CompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	JSONMode    bool      `json:"json_mode"`
}

// Message represents a single message in a conversation.
type Message struct {
	Role    string `json:"role"` // "system", "user", "assistant"
	Content string `json:"content"`
}

// RateLimitError indicates a provider returned HTTP 429 with an optional retry delay.
type RateLimitError struct {
	RetryAfter time.Duration
	Message    string
}

func (e *RateLimitError) Error() string {
	return e.Message
}

// CompletionResponse contains the result of a single LLM call.
type CompletionResponse struct {
	Content      string `json:"content"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	Model        string `json:"model"`
}
