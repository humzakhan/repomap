package planner

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed models.json
var modelsJSON []byte

// ModelCatalog contains all available models and their pricing.
type ModelCatalog struct {
	UpdatedAt string        `json:"updated_at"`
	Models    []ModelConfig `json:"models"`
}

// ModelConfig describes a single LLM model's capabilities and pricing.
type ModelConfig struct {
	ID                   string   `json:"id"`
	DisplayName          string   `json:"display_name"`
	Provider             string   `json:"provider"`
	ContextWindow        int      `json:"context_window"`
	InputCostPerMillion  float64  `json:"input_cost_per_million"`
	OutputCostPerMillion float64  `json:"output_cost_per_million"`
	QualityTier          string   `json:"quality_tier"` // "economy", "balanced", "frontier"
	RecommendedFor       []string `json:"recommended_for"`
	Notes                *string  `json:"notes"`
}

// LoadModels parses the embedded models.json.
func LoadModels() (*ModelCatalog, error) {
	var catalog ModelCatalog
	decoder := json.NewDecoder(strings.NewReader(string(modelsJSON)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&catalog); err != nil {
		return nil, fmt.Errorf("parsing models.json: %w", err)
	}
	return &catalog, nil
}

// ForProvider returns all models from a specific provider.
func (c *ModelCatalog) ForProvider(provider string) []ModelConfig {
	var result []ModelConfig
	for _, m := range c.Models {
		if m.Provider == provider {
			result = append(result, m)
		}
	}
	return result
}

// FindByID returns a model by its ID, or nil if not found.
func (c *ModelCatalog) FindByID(id string) *ModelConfig {
	for i := range c.Models {
		if c.Models[i].ID == id {
			return &c.Models[i]
		}
	}
	return nil
}

// Providers returns a deduplicated list of all provider names.
func (c *ModelCatalog) Providers() []string {
	seen := make(map[string]bool)
	var providers []string
	for _, m := range c.Models {
		if !seen[m.Provider] {
			seen[m.Provider] = true
			providers = append(providers, m.Provider)
		}
	}
	return providers
}
