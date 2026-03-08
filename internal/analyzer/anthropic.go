package analyzer

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// AnthropicProvider implements the Provider interface for Anthropic's Claude models.
type AnthropicProvider struct {
	client *anthropic.Client
	apiKey string
}

// NewAnthropicProvider creates a new Anthropic provider with the given API key.
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	client := anthropic.NewClient(option.WithAPIKey(apiKey))
	return &AnthropicProvider{
		client: &client,
		apiKey: apiKey,
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	messages := make([]anthropic.MessageParam, 0, len(req.Messages))
	var systemPrompt string

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			systemPrompt = msg.Content
		case "user":
			messages = append(messages, anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		case "assistant":
			messages = append(messages, anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			))
		}
	}

	maxTokens := int64(req.MaxTokens)
	if maxTokens == 0 {
		maxTokens = 4096
	}

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(req.Model),
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}

	resp, err := p.client.Messages.New(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("anthropic completion: %w", err)
	}

	var content string
	for _, block := range resp.Content {
		if block.Type == "text" {
			content += block.Text
		}
	}

	return &CompletionResponse{
		Content:      content,
		InputTokens:  int(resp.Usage.InputTokens),
		OutputTokens: int(resp.Usage.OutputTokens),
		Model:        string(resp.Model),
	}, nil
}

func (p *AnthropicProvider) EstimateCost(inputTokens, outputTokens int, modelID string) float64 {
	// Pricing per million tokens (as of March 2026)
	pricing := map[string][2]float64{
		"claude-haiku-3-5": {0.80, 4.00},
		"claude-sonnet-4":  {3.00, 15.00},
	}

	rates, ok := pricing[modelID]
	if !ok {
		return 0
	}

	inputCost := float64(inputTokens) / 1_000_000 * rates[0]
	outputCost := float64(outputTokens) / 1_000_000 * rates[1]
	return inputCost + outputCost
}
