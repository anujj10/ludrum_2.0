package signalengine

import "ludrum/internal/models"

func EnrichPairSignals(pairs []models.PairSignal) []models.PairSignal {

	for i := range pairs {

		signal := DetectDirectionalSignal(pairs[i])

		// Override bias with smarter logic
		pairs[i].Bias = string(signal.Type)

		// Convert strength → score (0–100)
		pairs[i].Score = signal.Strength * 100
	}

	return pairs
}