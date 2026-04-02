package payloadbuilder

import "ludrum/internal/models"

func buildMarket(pairs []models.PairSignal) MarketState {

	if len(pairs) == 0 {
		return MarketState{}
	}

	totalScore := 0.0
	bullish := 0

	for _, p := range pairs {
		totalScore += p.Score

		if p.Bias == "STRONG_BULLISH" || p.Bias == "WEAK_BULLISH" {
			bullish++
		}
	}

	avgScore := totalScore / float64(len(pairs))

	bias := "NEUTRAL"
	if bullish > len(pairs)/2 {
		bias = "BULLISH"
	} else {
		bias = "BEARISH"
	}

	strength := int((avgScore / 5.0) * 100)
	if strength > 100 {
		strength = 100
	}

	regime := "SIDEWAYS"
	if strength > 60 {
		regime = "TRENDING"
	}

	confidence := float64(strength) / 100.0

	return MarketState{
		Bias:       bias,
		Strength:   strength,
		Regime:     regime,
		Confidence: confidence,
	}
}