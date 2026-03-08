package planner

import "github.com/repomap/repomap/internal/scanner"

// CostEstimate contains the estimated cost for a single model.
type CostEstimate struct {
	Model      ModelConfig `json:"model"`
	InputCost  float64     `json:"input_cost"`
	OutputCost float64     `json:"output_cost"`
	Total      float64     `json:"total"`
	FitsContext bool       `json:"fits_context"`
	Warning    string      `json:"warning,omitempty"`
}

// EstimateCost calculates the cost of running an analysis with a specific model.
func EstimateCost(model ModelConfig, budget scanner.TokenBudget) CostEstimate {
	inputCost := float64(budget.TotalInput) / 1_000_000 * model.InputCostPerMillion
	outputCost := float64(budget.EstimatedOutput) / 1_000_000 * model.OutputCostPerMillion

	fits := budget.TotalInput <= model.ContextWindow
	warning := ""
	if !fits {
		warning = "⚠ small ctx"
	}

	return CostEstimate{
		Model:       model,
		InputCost:   inputCost,
		OutputCost:  outputCost,
		Total:       inputCost + outputCost,
		FitsContext:  fits,
		Warning:     warning,
	}
}

// EstimateAllModels calculates cost estimates for every model in the catalog.
func EstimateAllModels(catalog *ModelCatalog, budget scanner.TokenBudget) []CostEstimate {
	estimates := make([]CostEstimate, 0, len(catalog.Models))
	for _, model := range catalog.Models {
		estimates = append(estimates, EstimateCost(model, budget))
	}
	return estimates
}
