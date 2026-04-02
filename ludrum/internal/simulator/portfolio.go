package simulator

type Portfolio struct {
	InitialCapital   float64
	AvailableCapital float64
	UsedMargin       float64

	RealizedPnL   float64
	UnrealizedPnL float64
}

func NewPortfolio(capital float64) *Portfolio {
	return &Portfolio{
		InitialCapital:   capital,
		AvailableCapital: capital,
	}
}

func (p *Portfolio) CanTakeMargin(margin float64) bool {
	return p.AvailableCapital >= margin
}

func (p *Portfolio) BlockMargin(margin float64) {
	p.AvailableCapital -= margin
	p.UsedMargin += margin
}

func (p *Portfolio) ReleaseMargin(margin float64) {
	p.AvailableCapital += margin
	p.UsedMargin -= margin
}