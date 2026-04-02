package types

import "time"

type DBSnapshot struct {
	Time        time.Time
	Symbol      string
	SpotPrice   float64
	VIX         float64
	TotalCallOI int64
	TotalPutOI  int64
}

type DBOption struct {
	Time       time.Time
	Symbol     string
	Strike     float64
	OptionType string // "CE" or "PE"

	LTP float64
	Bid float64
	Ask float64

	OI          int64
	OIChange    int64
	OIChangePct float64
	Volume      int64
}

type DBFeature struct {
	Time       time.Time
	Symbol     string
	Strike     float64
	OptionType string

	OI       int64
	OIChange int64
	Volume   int64

	OIVelocity   float64
	VolumeSpike  float64
	PriceChange  float64
	BidAskSpread float64

	SpotPrice       float64
	DistanceFromATM float64
}

type DBOIChangeEvent struct {
	Time       time.Time
	Symbol     string
	Strike     float64
	OptionType string
	OIChange   int64
	LTPChange  float64
}
