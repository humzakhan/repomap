package planner

import "sort"

// ScoredModel extends CostEstimate with a recommendation score.
type ScoredModel struct {
	CostEstimate
	Score         float64 `json:"score"`
	IsRecommended bool    `json:"is_recommended"`
	IsConnected   bool    `json:"is_connected"`
}

// qualityTierRank maps quality tiers to numeric ranks for comparison.
var qualityTierRank = map[string]int{
	"economy":  1,
	"balanced": 2,
	"frontier": 3,
}

// ScoreModels ranks all models and selects the recommended one.
// The recommendation goes to the cheapest model that:
// 1. Fits the full token budget within its context window
// 2. Has quality_tier of "balanced" or above
// 3. Is from a connected provider
func ScoreModels(estimates []CostEstimate, connectedProviders []string) []ScoredModel {
	connected := make(map[string]bool)
	for _, p := range connectedProviders {
		connected[p] = true
	}

	scored := make([]ScoredModel, 0, len(estimates))
	for _, est := range estimates {
		sm := ScoredModel{
			CostEstimate: est,
			IsConnected:  connected[est.Model.Provider],
		}

		// Score: lower is better (cost-based), with bonuses
		sm.Score = est.Total

		// Penalty for not fitting context
		if !est.FitsContext {
			sm.Score += 100
		}

		// Penalty for economy tier
		tier := qualityTierRank[est.Model.QualityTier]
		if tier < 2 {
			sm.Score += 50
		}

		// Penalty for not being connected
		if !sm.IsConnected {
			sm.Score += 200
		}

		scored = append(scored, sm)
	}

	// Sort: connected first, then by score ascending
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].IsConnected != scored[j].IsConnected {
			return scored[i].IsConnected
		}
		return scored[i].Score < scored[j].Score
	})

	// Find the recommended model: cheapest connected model that fits context
	// and has quality_tier >= "balanced"
	for i := range scored {
		if scored[i].IsConnected &&
			scored[i].FitsContext &&
			qualityTierRank[scored[i].Model.QualityTier] >= 2 {
			scored[i].IsRecommended = true
			break
		}
	}

	return scored
}
