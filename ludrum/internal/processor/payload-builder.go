package processor

import (
	"strings"
	"time"

	"ludrum/internal/models"
)

// ==========================
// 🔥 SYMBOL NORMALIZER
// ==========================
func normalizeSymbol(sym string) string {
	if strings.Contains(sym, "NIFTY") {
		return "NIFTY"
	}
	return sym
}

// ==========================
// 🔥 BUILD FROM TICK (WS)
// ==========================
func BuildPayloadFromTick(t models.MarketTick) models.Payload {

	return models.Payload{
		Market: models.MarketInfo{
			Symbol:        normalizeSymbol(t.Symbol),
			LTP:           t.LTP,
			Change:        t.Change,
			ChangePercent: t.ChangePercent,
			Bias:          "NEUTRAL", // placeholder (future engine)
			Strength:      0,
		},
		Signals: map[string]interface{}{}, // empty for now

		Meta: models.MetaInfo{
			Timestamp: time.Now().Unix(),
			Source:    "ws",
		},
	}
}

// ==========================
// 🔥 BUILD FROM SNAPSHOT (REST)
// ==========================
func BuildPayloadFromSnapshot(snap *models.MarketSnapshot) models.Payload {

	return models.Payload{
		Market: models.MarketInfo{
			Symbol:        normalizeSymbol(snap.Symbol),
			LTP:           snap.SpotPrice,
			Change:        0, // fill later if needed
			ChangePercent: 0,
			Bias:          "NEUTRAL",
			Strength:      0,
		},
		Signals: map[string]interface{}{},

		Meta: models.MetaInfo{
			Timestamp: snap.Timestamp,
			Source:    "rest",
		},
	}
}

func BuildPayloadFromTickWithAlpha(
	t models.MarketTick,
	alpha AlphaSignals,
) models.Payload {

	return models.Payload{
		Market: models.MarketInfo{
			Symbol:        normalizeSymbol(t.Symbol),
			LTP:           t.LTP,
			Change:        t.Change,
			ChangePercent: t.ChangePercent,
			Bias:          "NEUTRAL",
			Strength:      0,
		},

		Signals: map[string]interface{}{
			"ltp_sequence": alpha.LTPSequence,
			"velocity":     alpha.Velocity,
			"trend":        alpha.Trend,
		},

		Meta: models.MetaInfo{
			Timestamp: t.Timestamp,
			Source:    "ws",
		},
	}
}