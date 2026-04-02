package exitengine

import "ludrum/internal/models"

func DetectExit(
	pair models.PairSignal,
	entry float64,
	current float64,
	sl float64,
	target float64,
) ExitDecision {

	var a models.StrikeAnalytics

	// pick correct leg
	if pair.Bias == "STRONG_BULLISH" {
		a = pair.CE
	} else if pair.Bias == "STRONG_BEARISH" {
		a = pair.PE
	} else {
		return ExitDecision{
			Exit: false,
			Type: Hold,
		}
	}

	// =========================
	// 🔴 HARD EXIT CONDITIONS
	// =========================

	if current <= sl {
		return ExitDecision{
			Exit:   true,
			Type:   StopLossHit,
			Reason: "Stop loss hit",
		}
	}

	if current >= target {
		return ExitDecision{
			Exit:   true,
			Type:   TargetHit,
			Reason: "Target achieved",
		}
	}

	// =========================
	// ⚠️ MOMENTUM FADE EXIT
	// =========================

	if isMomentumFading(a) {
		return ExitDecision{
			Exit:   true,
			Type:   MomentumExit,
			Reason: "Momentum fading",
		}
	}

	// =========================
	// ⚠️ TRAILING EXIT
	// =========================

	if current > entry*1.2 && isWeakContinuation(a) {
		return ExitDecision{
			Exit:   true,
			Type:   TrailingExit,
			Reason: "Weak continuation after profit",
		}
	}

	return ExitDecision{
		Exit: false,
		Type: Hold,
	}
}