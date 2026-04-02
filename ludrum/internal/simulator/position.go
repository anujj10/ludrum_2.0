package simulator

import "time"

type PositionSide string

const (
	LONG  PositionSide = "LONG"
	SHORT PositionSide = "SHORT"
)

type Position struct {
	Symbol     string
	Strike     float64
	OptionType string

	Qty      int
	AvgPrice float64
	Side     PositionSide

	SL     *float64
	Target *float64

	UnrealizedPnL float64
	RealizedPnL   float64

	EntryTime  int64
	LastUpdate int64
}

func (p *Position) UpdatePnL(currentPrice float64) {
	if p.Side == LONG {
		p.UnrealizedPnL = (currentPrice - p.AvgPrice) * float64(p.Qty)
	} else {
		p.UnrealizedPnL = (p.AvgPrice - currentPrice) * float64(p.Qty)
	}
	p.LastUpdate = time.Now().Unix()
}