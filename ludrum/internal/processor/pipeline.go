package processor

import (
	"log"

	signalengine "ludrum/internal/engine/signal-engine"
	"ludrum/internal/models"
	"ludrum/internal/simulator"
)

type Pipeline struct {
	Engine          *Engine
	StrikeRange     float64
	CleanupRange    float64
	ChangeDetector  *ChangeDetector
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		Engine:         NewEngine(),
		ChangeDetector: NewChangeDetector(),
		StrikeRange:    200,
		CleanupRange:   400,
	}
}

func (p *Pipeline) Process(
	snap *models.MarketSnapshot,
	sim *simulator.Simulator,
) models.StreamPayload {

	// ==========================
	// CHANGE DETECTION
	// ==========================
	changed := p.ChangeDetector.Filter(snap, p.StrikeRange)
	log.Println("Changed strikes:", len(changed))

	// ==========================
	// ENGINE
	// ==========================
	p.Engine.AddChangedTicks(changed)

	analytics := p.Engine.AnalyzeChanged(changed)

	// ==========================
	// SIGNALS
	// ==========================
	pairs := BuildPairSignals(analytics, snap.SpotPrice)
	pairs = signalengine.EnrichPairSignals(pairs)
			if pairs == nil{
			pairs = make([]models.PairSignal, 0)
		}

	cross := AnalyzeCrossStrike(pairs)

	log.Printf(
		"CROSS → Bullish: %v | Bearish: %v | StrongCE: %.0f | StrongPE: %.0f",
		cross.BullishShift,
		cross.BearishShift,
		cross.StrongestCE,
		cross.StrongestPE,
	)

	// ==========================
	// SIMULATOR
	// ==========================
	for _, pair := range pairs {

		if len(pair.CE.LTPSeries) > 0 {
			price := pair.CE.LTPSeries[len(pair.CE.LTPSeries)-1]
			sim.OnPriceUpdate(pair.Strike, "CE", price)
		}

		if len(pair.PE.LTPSeries) > 0 {
			price := pair.PE.LTPSeries[len(pair.PE.LTPSeries)-1]
			sim.OnPriceUpdate(pair.Strike, "PE", price)
		}
	}

	openPos, closedPos, portfolio := sim.GetState()

	// ==========================
	// CLEANUP
	// ==========================
	if p.Engine.tickCount%20 == 0 {
		p.Engine.Cleanup(snap.SpotPrice, p.CleanupRange)
	}

	// ==========================
	// PAYLOAD TYPE (IMPORTANT)
	// ==========================
	payloadType := "delta"
	if p.Engine.tickCount == 1 || p.Engine.tickCount%5 == 0 {
		payloadType = "snapshot"
	}


	// ==========================
	// FINAL PAYLOAD
	// ==========================
	payload := models.StreamPayload{
		Type: payloadType, // 🔥 IMPORTANT
		Spot: snap.SpotPrice,
		Pairs: pairs,

		OpenPositions:   openPos,
		ClosedPositions: closedPos,
		Portfolio:       portfolio,
	}

	return payload
}
