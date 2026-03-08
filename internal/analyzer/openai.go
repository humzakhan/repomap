package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// OpenAIProvider implements the Provider interface for OpenAI models.
// Also used as base for OpenAI-compatible APIs (Groq, Kimi).
type OpenAIProvider struct {
	name    string
	apiKey  string
	baseURL string
	client  *http.Client
}

// NewOpenAIProvider creates a new OpenAI provider.
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		name:    "openai",
		apiKey:  apiKey,
		baseURL: "https://api.openai.com/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// NewGroqProvider creates a Groq provider using the OpenAI-compatible API.
func NewGroqProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		name:    "groq",
		apiKey:  apiKey,
		baseURL: "https://api.groq.com/openai/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

// NewKimiProvider creates a Kimi (Moonshot) provider using the OpenAI-compatible API.
func NewKimiProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		name:    "kimi",
		apiKey:  apiKey,
		baseURL: "https://api.moonshot.cn/v1",
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *OpenAIProvider) Name() string {
	return p.name
}

type openAIRequest struct {
	Model       string          `json:"model"`
	Messages    []openAIMessage `json:"messages"`
	Temperature float64         `json:"temperature"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	ResponseFormat *openAIResponseFormat `json:"response_format,omitempty"`
}

type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIResponseFormat struct {
	Type string `json:"type"`
}

type openAIResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

func (p *OpenAIProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	messages := make([]openAIMessage, 0, len(req.Messages))
	for _, msg := range req.Messages {
		messages = append(messages, openAIMessage{
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	oaiReq := openAIRequest{
		Model:       req.Model,
		Messages:    messages,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	if req.JSONMode {
		oaiReq.ResponseFormat = &openAIResponseFormat{Type: "json_object"}
	}

	body, err := json.Marshal(oaiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling %s request: %w", p.name, err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating %s request: %w", p.name, err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("%s API call: %w", p.name, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading %s response: %w", p.name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s API error (HTTP %d): %s", p.name, resp.StatusCode, string(respBody))
	}

	var oaiResp openAIResponse
	if err := json.Unmarshal(respBody, &oaiResp); err != nil {
		return nil, fmt.Errorf("parsing %s response: %w", p.name, err)
	}

	if oaiResp.Error != nil {
		return nil, fmt.Errorf("%s API error: %s", p.name, oaiResp.Error.Message)
	}

	if len(oaiResp.Choices) == 0 {
		return nil, fmt.Errorf("%s returned no choices", p.name)
	}

	return &CompletionResponse{
		Content:      oaiResp.Choices[0].Message.Content,
		InputTokens:  oaiResp.Usage.PromptTokens,
		OutputTokens: oaiResp.Usage.CompletionTokens,
		Model:        oaiResp.Model,
	}, nil
}

func (p *OpenAIProvider) EstimateCost(inputTokens, outputTokens int, modelID string) float64 {
	pricing := map[string][2]float64{
		"gpt-4o-mini":  {0.15, 0.60},
		"gpt-4o":       {2.50, 10.00},
		"llama-3.1-70b": {0.59, 0.79},
		"kimi-k2":      {0.60, 2.50},
		"kimi-k2.5":    {0.60, 3.00},
	}

	rates, ok := pricing[modelID]
	if !ok {
		return 0
	}

	inputCost := float64(inputTokens) / 1_000_000 * rates[0]
	outputCost := float64(outputTokens) / 1_000_000 * rates[1]
	return inputCost + outputCost
}
