package scoringengine

import (
	entryengine "ludrum/internal/engine/entry-engine"
	"ludrum/internal/models"
)

func ScoreTrade(pair models.PairSignal) ScoreResult {

	score := 0.0
	var reasons []string

	ce := pair.CE
	pe := pair.PE

	// =========================
	// 1. VOLUME
	// =========================

	if ce.VolumeChange >= 800000 || pe.VolumeChange >= 800000 {
		score += VolumeWeight
		reasons = append(reasons, "volume spike")
	}

	// =========================
	// 2. OI STRUCTURE
	// =========================

	if pair.Bias == "STRONG_BULLISH" && ce.OIChange > 0 {
		score += OIWeight
		reasons = append(reasons, "ce oi buildup")
	}

	if pair.Bias == "STRONG_BEARISH" && pe.OIChange > 0 {
		score += OIWeight
		reasons = append(reasons, "pe oi buildup")
	}

	// =========================
	// 3. LTP DOMINANCE
	// =========================

	if pair.Bias == "STRONG_BULLISH" {
		if isStrongMove(ce) {
			score += LTPWeight
			reasons = append(reasons, "ce momentum strong")
		}
	}

	if pair.Bias == "STRONG_BEARISH" {
		if isStrongMove(pe) {
			score += LTPWeight
			reasons = append(reasons, "pe momentum strong")
		}
	}

	// =========================
	// 4. STRUCTURE QUALITY
	// =========================

	if pair.Score > 80 {
		score += StructureWeight
		reasons = append(reasons, "structure aligned")
	}

	// =========================
	// 5. ENTRY TIMING
	// =========================

	entry := entryengine.DetectEntry(pair)

	if entry.Valid {
		score += EntryTimingWeight
		reasons = append(reasons, "good entry timing")
	}

	// =========================
	// FINAL
	// =========================

	confidence := score / 100.0

	return ScoreResult{
		Score:      score,
		Confidence: confidence,
		Reasons:    reasons,
	}
}

// helper
func isStrongMove(a models.StrikeAnalytics) bool {
	if len(a.LTPDeltas) < 2 {
		return false
	}

	n := len(a.LTPDeltas)

	return a.LTPDeltas[n-1] > 0 && a.LTPDeltas[n-2] > 0
}