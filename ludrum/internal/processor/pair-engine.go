package processor

import (
	"ludrum/internal/models"
	"math"
)

func getATMStrike(spot float64) float64 {
	return math.Round(spot/50) * 50
}

// ==========================
// FULL PAIR ANALYSIS (CE + PE)
// ==========================
func analyzePair(ce, pe models.StrikeAnalytics) (string, float64) {

	score := 0.0

	// --- CE Logic ---
	if ce.LTPChange > 0 {
		score += 1
	}
	if ce.OIChange > 0 {
		score += 1
	}
	if ce.VolumeChange > 0 {
		score += 0.5
	}

	// --- PE Logic ---
	if pe.LTPChange < 0 {
		score += 1
	}
	if pe.OIChange < 0 {
		score += 1
	}
	if pe.VolumeChange > 0 {
		score += 0.5
	}

	// --- Bias Decision ---
	switch {
	case score >= 4:
		return "STRONG_BULLISH", score
	case score >= 2.5:
		return "WEAK_BULLISH", score
	case score <= -3:
		return "STRONG_BEARISH", score
	case score <= -1.5:
		return "WEAK_BEARISH", score
	default:
		return "NEUTRAL", score
	}
}

// ==========================
// SINGLE SIDE ANALYSIS (FIX)
// ==========================
func analyzeSingleSide(a models.StrikeAnalytics, side string) (string, float64) {

	score := 0.0

	if a.LTPChange > 0 {
		score += 1
	}
	if a.OIChange > 0 {
		score += 1
	}
	if a.VolumeChange > 0 {
		score += 0.5
	}

	if side == "CE" {
		switch {
		case score >= 2:
			return "CE_BULLISH", score
		case score >= 1:
			return "CE_WEAK_BULLISH", score
		default:
			return "CE_NEUTRAL", score
		}
	}

	if side == "PE" {
		switch {
		case score >= 2:
			return "PE_BEARISH", score
		case score >= 1:
			return "PE_WEAK_BEARISH", score
		default:
			return "PE_NEUTRAL", score
		}
	}

	return "NEUTRAL", score
}

// ==========================
// BUILD PAIRS (FIXED)
// ==========================
func BuildPairSignals(
	analytics []models.StrikeAnalytics,
	spot float64,
) []models.PairSignal {

	atm := getATMStrike(spot)

	// Map for quick lookup
	strikeMap := make(map[float64]map[string]models.StrikeAnalytics)

	for _, a := range analytics {
		if _, ok := strikeMap[a.Strike]; !ok {
			strikeMap[a.Strike] = make(map[string]models.StrikeAnalytics)
		}
		strikeMap[a.Strike][a.Type] = a
	}

	var results []models.PairSignal

	// 🔥 Adaptive range (IMPORTANT FIX)
	rangeLimit := 100.0
	if len(strikeMap) < 10 {
		rangeLimit = 200
	}

	// Focus around ATM
	for strike, data := range strikeMap {

		if math.Abs(strike-atm) > rangeLimit {
			continue
		}

		ce, okCE := data["CE"]
		pe, okPE := data["PE"]

		if !okCE && !okPE {
			continue
		}

		var bias string
		var score float64

		// 🔥 FIX: HANDLE PARTIAL DATA
		if okCE && okPE {
			bias, score = analyzePair(ce, pe)

		} else if okCE {
			bias, score = analyzeSingleSide(ce, "CE")

		} else if okPE {
			bias, score = analyzeSingleSide(pe, "PE")
		}

		// 🔥 FIX: ONLY DROP TRULY EMPTY SIGNALS
		if score == 0 {
			continue
		}

		results = append(results, models.PairSignal{
			Strike: strike,
			CE:     ce,
			PE:     pe,
			Bias:   bias,
			Score:  score,
		})
	}

	return results
}