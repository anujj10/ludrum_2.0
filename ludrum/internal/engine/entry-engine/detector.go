package entryengine

import "ludrum/internal/models"

func DetectEntry(pair models.PairSignal) EntryDecision {

	ce := pair.CE

	// =========================
	// 🟢 BULLISH ENTRY
	// =========================

	if pair.Bias == "STRONG_BULLISH" {

		// ✅ Ideal pullback entry
		if isBullishPullback(ce) {
			return EntryDecision{
				Valid:  true,
				Type:   PullbackEntry,
				Reason: "Pullback + continuation confirmed",
			}
		}

		// ❌ Avoid chasing
		if isOverExtended(ce) {
			return EntryDecision{
				Valid:  false,
				Type:   NoEntry,
				Reason: "Overextended move (avoid chasing)",
			}
		}
	}

	// =========================
	// 🔴 BEARISH ENTRY (mirror)
	// =========================

	if pair.Bias == "STRONG_BEARISH" {

		// mirror logic using PE
		if isBullishPullback(pair.PE) {
			return EntryDecision{
				Valid:  true,
				Type:   PullbackEntry,
				Reason: "Bearish pullback continuation",
			}
		}
	}

	return EntryDecision{
		Valid:  false,
		Type:   NoEntry,
		Reason: "No clear structure",
	}
}