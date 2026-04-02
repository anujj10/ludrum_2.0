package payloadbuilder

import "ludrum/internal/models"

type Builder struct {
	capital float64
	riskPct float64
}

func NewBuilder(capital float64, riskPct float64) *Builder {
	return &Builder{
		capital: capital,
		riskPct: riskPct,
	}
}

func (b *Builder) Build(pairs []models.PairSignal, snap models.MarketSnapshot) FinalPayload {

	market := buildMarket(pairs)
	signals := buildSignals(pairs)
	trade := buildTrade(pairs, snap.Symbol)
	risk := b.buildRisk(trade)

	return FinalPayload{
		Market:    market,
		Signals:   signals,
		Trade:     trade,
		Risk:      risk,
		Positions: []Position{}, // empty for now (live mode)
	}
}