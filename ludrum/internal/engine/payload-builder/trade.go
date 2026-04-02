package payloadbuilder

import (
	exitengine "ludrum/internal/engine/exit-engine"
	scoringengine "ludrum/internal/engine/score-engine"
	"ludrum/internal/models"
)

func pickBestPair(pairs []models.PairSignal) *models.PairSignal {

	if len(pairs) == 0 {
		return nil
	}

	best := &pairs[0]

	for i := range pairs {
		if pairs[i].Score > best.Score {
			best = &pairs[i]
		}
	}

	return best
}

func buildTrade(pairs []models.PairSignal, symbol string) TradeDecision {

	best := pickBestPair(pairs)

	if best == nil {
		return TradeDecision{Action: "HOLD"}
	}

	// 🔥 ENTRY ENGINE
scoreResult := scoringengine.ScoreTrade(*best)

// ❌ LOW QUALITY TRADE → SKIP
if scoreResult.Score < 60 {
	return TradeDecision{
		Action: "HOLD",
	}
}

	var optionType string
	var entry float64

	if best.Bias == "STRONG_BULLISH" {
		optionType = "CE"
		entry = best.CE.LTPSeries[len(best.CE.LTPSeries)-1]
	} else if best.Bias == "STRONG_BEARISH" {
		optionType = "PE"
		entry = best.PE.LTPSeries[len(best.PE.LTPSeries)-1]
	} else {
		return TradeDecision{Action: "HOLD"}
	}

	sl := entry * 0.8
	target := entry * 1.5

	current := entry // ⚠️ for now (later connect live price)

	// 🔥 EXIT ENGINE
	exitDecision := exitengine.DetectExit(*best, entry, current, sl, target)

	if exitDecision.Exit {
		return TradeDecision{
			Action: "EXIT",
			Symbol: symbol,
			Strike: best.Strike,
		}
	}

	return TradeDecision{
		Action:     "BUY",
		Symbol:     symbol,
		Strike:     best.Strike,
		OptionType: optionType,
		Entry:      entry,
		SL:         sl,
		Target:     target,
		RR:         2.0,
		Score:      scoreResult.Score,
		Confidence: scoreResult.Confidence,
		Reasons:    scoreResult.Reasons,
	}
}