package ltpSeries

import "fmt"

type StrikeSelector struct {
	rangeCount int
	expiry     string
	step       int
}

func NewStrikeSelector(rangeCount int, expiry string, step int) *StrikeSelector {
	return &StrikeSelector{
		rangeCount: rangeCount,
		expiry:     expiry,
		step:       step,
	}
}

func (s *StrikeSelector) GenerateSymbols(atm float64) []string {
	var symbols []string

	for i := -s.rangeCount; i <= s.rangeCount; i++ {
		strike := int(atm) + (i * s.step)

		ce := fmt.Sprintf("NSE:NIFTY%s%dCE", s.expiry, strike)
		pe := fmt.Sprintf("NSE:NIFTY%s%dPE", s.expiry, strike)

		symbols = append(symbols, ce, pe)
	}

	return symbols
}