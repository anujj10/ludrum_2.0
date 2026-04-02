package ltpSeries

import (
	"context"
	"log"
)

// ==========================
// ENGINE (CONNECTS EVERYTHING)
// ==========================
type LTPEngine struct {
	atmTracker     *ATMTracker
	strikeSelector *StrikeSelector
	poller         *LTPPoller
}

// ==========================
// CONSTRUCTOR
// ==========================
func NewLTPEngine(
	atm *ATMTracker,
	selector *StrikeSelector,
	poller *LTPPoller,
) *LTPEngine {
	return &LTPEngine{
		atmTracker:     atm,
		strikeSelector: selector,
		poller:         poller,
	}
}

// ==========================
// START ENGINE
// ==========================
func (e *LTPEngine) Start(ctx context.Context) {
	go e.poller.Start(ctx)
}

// ==========================
// ON INDEX TICK (CALL FROM WS)
// ==========================
func (e *LTPEngine) OnIndexTick(ltp float64) {

	atm, changed := e.atmTracker.Update(ltp)

	if !changed {
		return
	}

	symbols := e.strikeSelector.GenerateSymbols(atm)

	log.Println("New ATM:", atm, "Tracking symbols:", symbols)

	e.poller.UpdateTrackedSymbols(symbols)
}