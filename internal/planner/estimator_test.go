package planner

import (
	"math"
	"testing"

	"github.com/repomap/repomap/internal/scanner"
)

func TestEstimateCostAllModels(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	budget := scanner.TokenBudget{
		Summarization:   100000,
		Synthesis:       15000,
		DocIngestion:    10000,
		TotalInput:      125000,
		EstimatedOutput: 15625,
	}

	for _, model := range catalog.Models {
		t.Run(model.ID, func(t *testing.T) {
			est := EstimateCost(model, budget)

			if est.InputCost <= 0 {
				t.Errorf("expected positive input cost, got %f", est.InputCost)
			}
			if est.OutputCost <= 0 {
				t.Errorf("expected positive output cost, got %f", est.OutputCost)
			}
			if math.Abs(est.Total-(est.InputCost+est.OutputCost)) > 0.001 {
				t.Errorf("total %f != input %f + output %f", est.Total, est.InputCost, est.OutputCost)
			}

			// Verify cost calculation formula
			expectedInput := float64(budget.TotalInput) / 1_000_000 * model.InputCostPerMillion
			expectedOutput := float64(budget.EstimatedOutput) / 1_000_000 * model.OutputCostPerMillion
			if math.Abs(est.InputCost-expectedInput) > 0.001 {
				t.Errorf("input cost %f != expected %f", est.InputCost, expectedInput)
			}
			if math.Abs(est.OutputCost-expectedOutput) > 0.001 {
				t.Errorf("output cost %f != expected %f", est.OutputCost, expectedOutput)
			}
		})
	}
}

func TestEstimateCostFitsContext(t *testing.T) {
	model := ModelConfig{
		ID:                   "test",
		ContextWindow:        100000,
		InputCostPerMillion:  1.0,
		OutputCostPerMillion: 2.0,
	}

	// Budget fits within context window
	fits := EstimateCost(model, scanner.TokenBudget{TotalInput: 50000, EstimatedOutput: 6250})
	if !fits.FitsContext {
		t.Error("expected budget to fit context window")
	}
	if fits.Warning != "" {
		t.Errorf("expected no warning, got %s", fits.Warning)
	}

	// Budget exceeds context window
	doesntFit := EstimateCost(model, scanner.TokenBudget{TotalInput: 150000, EstimatedOutput: 18750})
	if doesntFit.FitsContext {
		t.Error("expected budget to not fit context window")
	}
	if doesntFit.Warning == "" {
		t.Error("expected warning for small context")
	}
}

func TestEstimateAllModels(t *testing.T) {
	catalog, err := LoadModels()
	if err != nil {
		t.Fatalf("LoadModels failed: %v", err)
	}

	budget := scanner.TokenBudget{
		TotalInput:      50000,
		EstimatedOutput: 6250,
	}

	estimates := EstimateAllModels(catalog, budget)
	if len(estimates) != len(catalog.Models) {
		t.Errorf("expected %d estimates, got %d", len(catalog.Models), len(estimates))
	}
}

func TestEstimateCostSpecificValues(t *testing.T) {
	// Claude Haiku 3.5: $0.80/M input, $4.00/M output
	// 176000 input, 22000 output
	// Expected: 176000/1M * 0.80 = $0.1408 input
	//           22000/1M * 4.00 = $0.088 output
	//           Total: $0.2288
	model := ModelConfig{
		ID:                   "claude-haiku-3-5",
		ContextWindow:        200000,
		InputCostPerMillion:  0.80,
		OutputCostPerMillion: 4.00,
	}

	budget := scanner.TokenBudget{
		TotalInput:      176000,
		EstimatedOutput: 22000,
	}

	est := EstimateCost(model, budget)

	expectedInput := 0.1408
	expectedOutput := 0.088
	expectedTotal := 0.2288

	if math.Abs(est.InputCost-expectedInput) > 0.001 {
		t.Errorf("input cost: got %.4f, want %.4f", est.InputCost, expectedInput)
	}
	if math.Abs(est.OutputCost-expectedOutput) > 0.001 {
		t.Errorf("output cost: got %.4f, want %.4f", est.OutputCost, expectedOutput)
	}
	if math.Abs(est.Total-expectedTotal) > 0.001 {
		t.Errorf("total: got %.4f, want %.4f", est.Total, expectedTotal)
	}
}
