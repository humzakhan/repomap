package planner

import (
	"testing"
)

func TestLoadModels(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	if len(catalog.Models) == 0 {
		t.Fatal("expected at least one model")
	}

	if catalog.UpdatedAt == "" {
		t.Error("expected non-empty updated_at")
	}
}

func TestLoadModelsAllFieldsPresent(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	for _, m := range catalog.Models {
		if m.ID == "" {
			t.Error("model has empty ID")
		}
		if m.DisplayName == "" {
			t.Errorf("model %s has empty display_name", m.ID)
		}
		if m.Provider == "" {
			t.Errorf("model %s has empty provider", m.ID)
		}
		if m.ContextWindow <= 0 {
			t.Errorf("model %s has invalid context_window: %d", m.ID, m.ContextWindow)
		}
		if m.InputCostPerMillion <= 0 {
			t.Errorf("model %s has invalid input_cost_per_million: %f", m.ID, m.InputCostPerMillion)
		}
		if m.OutputCostPerMillion <= 0 {
			t.Errorf("model %s has invalid output_cost_per_million: %f", m.ID, m.OutputCostPerMillion)
		}
		if m.QualityTier == "" {
			t.Errorf("model %s has empty quality_tier", m.ID)
		}
		validTiers := map[string]bool{"economy": true, "balanced": true, "frontier": true}
		if !validTiers[m.QualityTier] {
			t.Errorf("model %s has invalid quality_tier: %s", m.ID, m.QualityTier)
		}
	}
}

func TestLoadModelsExpectedModels(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	expectedIDs := []string{
		"claude-haiku-3-5",
		"claude-sonnet-4",
		"gpt-4o-mini",
		"gpt-4o",
		"gemini-1.5-flash",
		"gemini-1.5-pro",
		"llama-3.1-70b",
	}

	for _, id := range expectedIDs {
		if catalog.FindByID(id) == nil {
			t.Errorf("expected model %s not found", id)
		}
	}
}

func TestForProvider(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	anthropicModels := catalog.ForProvider("anthropic")
	if len(anthropicModels) != 2 {
		t.Errorf("expected 2 anthropic models, got %d", len(anthropicModels))
	}

	openaiModels := catalog.ForProvider("openai")
	if len(openaiModels) != 2 {
		t.Errorf("expected 2 openai models, got %d", len(openaiModels))
	}

	googleModels := catalog.ForProvider("google")
	if len(googleModels) != 2 {
		t.Errorf("expected 2 google models, got %d", len(googleModels))
	}

	groqModels := catalog.ForProvider("groq")
	if len(groqModels) != 1 {
		t.Errorf("expected 1 groq model, got %d", len(groqModels))
	}
}

func TestFindByID(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	m := catalog.FindByID("claude-haiku-3-5")
	if m == nil {
		t.Fatal("expected to find claude-haiku-3-5")
	}
	if m.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", m.Provider)
	}
	if m.ContextWindow != 200000 {
		t.Errorf("expected context window 200000, got %d", m.ContextWindow)
	}

	missing := catalog.FindByID("nonexistent-model")
	if missing != nil {
		t.Error("expected nil for nonexistent model")
	}
}

func TestProviders(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	providers := catalog.Providers()
	expected := map[string]bool{
		"anthropic": false,
		"openai":    false,
		"google":    false,
		"groq":      false,
	}

	for _, p := range providers {
		expected[p] = true
	}

	for name, found := range expected {
		if !found {
			t.Errorf("expected provider %s not found", name)
		}
	}
}
