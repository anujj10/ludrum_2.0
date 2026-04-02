package processor

import "ludrum/internal/models"

type CrossStrikeSignal struct {
	BullishShift bool
	BearishShift bool

	StrongestCE float64
	StrongestPE float64
}

func AnalyzeCrossStrike(pairs []models.PairSignal) CrossStrikeSignal {

	var maxCEOI float64
	var maxPEOI float64

	var strongestCE float64
	var strongestPE float64

	totalCE := 0.0
	totalPE := 0.0

	for _, p := range pairs {

		// CE
		if p.CE.OIChange > 0 {
			totalCE += float64(p.CE.OIChange)

			if float64(p.CE.OIChange) > maxCEOI {
				maxCEOI = float64(p.CE.OIChange)
				strongestCE = p.Strike
			}
		}

		// PE
		if p.PE.OIChange > 0 {
			totalPE += float64(p.PE.OIChange)

			if float64(p.PE.OIChange) > maxPEOI {
				maxPEOI = float64(p.PE.OIChange)
				strongestPE = p.Strike
			}
		}
	}

	return CrossStrikeSignal{
		BullishShift: totalCE > totalPE,
		BearishShift: totalPE > totalCE,

		StrongestCE: strongestCE,
		StrongestPE: strongestPE,
	}
}