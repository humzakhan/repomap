package planner

import (
	"testing"

	"github.com/repomap/repomap/internal/scanner"
)

func TestScoreModelsRecommendation(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	// Budget that fits in 200K context but not 128K
	budget := scanner.TokenBudget{
		TotalInput:      150000,
		EstimatedOutput: 18750,
	}

	estimates := EstimateAllModels(catalog, budget)
	connected := []string{"anthropic", "openai"}

	scored := ScoreModels(estimates, connected)

	// Verify connected models come first
	seenUnconnected := false
	for _, sm := range scored {
		if !sm.IsConnected {
			seenUnconnected = true
		}
		if seenUnconnected && sm.IsConnected {
			t.Error("connected models should appear before unconnected")
		}
	}

	// Verify exactly one recommended model
	recommendedCount := 0
	var recommended *ScoredModel
	for i := range scored {
		if scored[i].IsRecommended {
			recommendedCount++
			recommended = &scored[i]
		}
	}
	if recommendedCount != 1 {
		t.Errorf("expected 1 recommended model, got %d", recommendedCount)
	}

	// Recommended should be connected, fit context, and be balanced+
	if recommended != nil {
		if !recommended.IsConnected {
			t.Error("recommended model should be connected")
		}
		if !recommended.FitsContext {
			t.Error("recommended model should fit context")
		}
		tier := qualityTierRank[recommended.Model.QualityTier]
		if tier < 2 {
			t.Errorf("recommended model should be balanced+, got %s", recommended.Model.QualityTier)
		}
	}
}

func TestScoreModelsNoConnected(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	budget := scanner.TokenBudget{
		TotalInput:      50000,
		EstimatedOutput: 6250,
	}

	estimates := EstimateAllModels(catalog, budget)
	scored := ScoreModels(estimates, nil) // no connected providers

	// No model should be recommended
	for _, sm := range scored {
		if sm.IsRecommended {
			t.Error("no model should be recommended when no providers are connected")
		}
	}
}

func TestScoreModelsSmallBudget(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	// Budget that fits all models
	budget := scanner.TokenBudget{
		TotalInput:      50000,
		EstimatedOutput: 6250,
	}

	estimates := EstimateAllModels(catalog, budget)
	connected := []string{"anthropic", "openai", "google", "groq"}

	scored := ScoreModels(estimates, connected)

	// All models should fit context
	for _, sm := range scored {
		if !sm.FitsContext {
			t.Errorf("model %s should fit context for 50K budget", sm.Model.ID)
		}
	}

	// Should have a recommendation
	hasRecommended := false
	for _, sm := range scored {
		if sm.IsRecommended {
			hasRecommended = true
			break
		}
	}
	if !hasRecommended {
		t.Error("expected a recommended model when all providers connected and budget fits")
	}
}

func TestScoreModelsConnectedFirst(t *testing.T) {
	estimates := []CostEstimate{
		{Model: ModelConfig{ID: "a", Provider: "x", QualityTier: "balanced"}, Total: 1.0, FitsContext: true},
		{Model: ModelConfig{ID: "b", Provider: "y", QualityTier: "balanced"}, Total: 0.5, FitsContext: true},
	}

	scored := ScoreModels(estimates, []string{"y"})

	if len(scored) != 2 {
		t.Fatalf("expected 2 scored models, got %d", len(scored))
	}
	if scored[0].Model.ID != "b" {
		t.Errorf("expected connected model 'b' first, got %s", scored[0].Model.ID)
	}
}
