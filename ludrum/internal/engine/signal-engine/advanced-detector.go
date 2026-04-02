package signalengine

import "ludrum/internal/models"

func DetectDirectionalSignal(pair models.PairSignal) SmartSignal {

	ce := pair.CE
	pe := pair.PE

	// =========================
	// CONDITIONS
	// =========================

	// volume filter (your rule)
	volumeOK := ce.VolumeChange >= 800000 || pe.VolumeChange >= 800000

	// LTP behavior
	ceUp := isLTPIncreasing(ce)
	peDown := isLTPDecreasingOrFlat(pe)

	peUp := isLTPIncreasing(pe)
	ceDown := isLTPDecreasingOrFlat(ce)

	// OI behavior
	ceOIUp := ce.OIChange > 0
	peOIUp := pe.OIChange > 0
	peOIDownOrFlat := pe.OIChange <= 0
	ceOIDownOrFlat := ce.OIChange <= 0

	// =========================
	// 🟢 STRONG BULLISH
	// =========================
	if volumeOK && ceUp && peDown && ceOIUp && peOIDownOrFlat {
		return SmartSignal{
			Type:       StrongBullish,
			Strength:   0.95,
			Confidence: 0.9,
		}
	}

	// =========================
	// 🔴 STRONG BEARISH
	// =========================
	if volumeOK && peUp && ceDown && peOIUp && ceOIDownOrFlat {
		return SmartSignal{
			Type:       StrongBearish,
			Strength:   0.95,
			Confidence: 0.9,
		}
	}

	// =========================
	// ⚠️ WEAK BULLISH
	// =========================
	if ceUp && ceOIUp {
		return SmartSignal{
			Type:       WeakBullish,
			Strength:   0.6,
			Confidence: 0.6,
		}
	}

	// =========================
	// ⚠️ WEAK BEARISH
	// =========================
	if peUp && peOIUp {
		return SmartSignal{
			Type:       WeakBearish,
			Strength:   0.6,
			Confidence: 0.6,
		}
	}

	return SmartSignal{
		Type:       Neutral,
		Strength:   0.3,
		Confidence: 0.3,
	}
}