package payloadbuilder

import "ludrum/internal/models"

func buildSignals(pairs []models.PairSignal) SignalBlock {

	if len(pairs) == 0 {
		return SignalBlock{}
	}

	// pick ATM (first one assumed closest)
	p := pairs[0]

	ceVel := float64(p.CE.OIChange)
	peVel := float64(p.PE.OIChange)

	return SignalBlock{
		OIVelocity: OIVelocity{
			CE: ceVel,
			PE: peVel,
		},
		LTPSeq: p.CE.LTPSeries,
	}
}