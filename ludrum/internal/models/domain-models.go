package models

// Core unit (already made)
type StrikeData struct {
	Strike float64

	CE *OptionChain
	PE *OptionChain
}

// Full market snapshot (THIS is your main working model)
type MarketSnapshot struct {
	Symbol     string
	SpotPrice  float64
	VIX        float64

	Strikes    map[float64]*StrikeData

	TotalCallOI int64
	TotalPutOI  int64

	Timestamp   int64
}