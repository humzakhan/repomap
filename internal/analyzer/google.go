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

// GoogleProvider implements the Provider interface for Google's Gemini models.
type GoogleProvider struct {
	apiKey string
	client *http.Client
}

// NewGoogleProvider creates a new Google AI provider.
func NewGoogleProvider(apiKey string) *GoogleProvider {
	return &GoogleProvider{
		apiKey: apiKey,
		client: &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *GoogleProvider) Name() string {
	return "google"
}

type googleRequest struct {
	Contents         []googleContent        `json:"contents"`
	SystemInstruction *googleContent         `json:"systemInstruction,omitempty"`
	GenerationConfig googleGenerationConfig `json:"generationConfig"`
}

type googleContent struct {
	Parts []googlePart `json:"parts"`
	Role  string       `json:"role,omitempty"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleGenerationConfig struct {
	Temperature    float64 `json:"temperature"`
	MaxOutputTokens int    `json:"maxOutputTokens,omitempty"`
	ResponseMimeType string `json:"responseMimeType,omitempty"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *GoogleProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var systemContent *googleContent
	var contents []googleContent

	for _, msg := range req.Messages {
		switch msg.Role {
		case "system":
			systemContent = &googleContent{
				Parts: []googlePart{{Text: msg.Content}},
			}
		case "user":
			contents = append(contents, googleContent{
				Role:  "user",
				Parts: []googlePart{{Text: msg.Content}},
			})
		case "assistant":
			contents = append(contents, googleContent{
				Role:  "model",
				Parts: []googlePart{{Text: msg.Content}},
			})
		}
	}

	gReq := googleRequest{
		Contents:          contents,
		SystemInstruction: systemContent,
		GenerationConfig: googleGenerationConfig{
			Temperature: req.Temperature,
		},
	}

	if req.MaxTokens > 0 {
		gReq.GenerationConfig.MaxOutputTokens = req.MaxTokens
	}

	if req.JSONMode {
		gReq.GenerationConfig.ResponseMimeType = "application/json"
	}

	body, err := json.Marshal(gReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling google request: %w", err)
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		req.Model, p.apiKey)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating google request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("google API call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading google response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google API error (HTTP %d): %s", resp.StatusCode, string(respBody))
	}

	var gResp googleResponse
	if err := json.Unmarshal(respBody, &gResp); err != nil {
		return nil, fmt.Errorf("parsing google response: %w", err)
	}

	if gResp.Error != nil {
		return nil, fmt.Errorf("google API error: %s", gResp.Error.Message)
	}

	if len(gResp.Candidates) == 0 || len(gResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("google returned no content")
	}

	var content string
	for _, part := range gResp.Candidates[0].Content.Parts {
		content += part.Text
	}

	return &CompletionResponse{
		Content:      content,
		InputTokens:  gResp.UsageMetadata.PromptTokenCount,
		OutputTokens: gResp.UsageMetadata.CandidatesTokenCount,
		Model:        req.Model,
	}, nil
}

func (p *GoogleProvider) EstimateCost(inputTokens, outputTokens int, modelID string) float64 {
	pricing := map[string][2]float64{
		"gemini-2.5-flash": {0.30, 2.50},
		"gemini-2.5-pro":   {1.25, 10.00},
	}

	rates, ok := pricing[modelID]
	if !ok {
		return 0
	}

	inputCost := float64(inputTokens) / 1_000_000 * rates[0]
	outputCost := float64(outputTokens) / 1_000_000 * rates[1]
	return inputCost + outputCost
}
