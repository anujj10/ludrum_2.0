package processor

import (
	"math"
)

// ==========================
// ALPHA OUTPUT
// ==========================
type AlphaSignals struct {
	LTPSequence []float64 `json:"ltp_sequence"`

	Velocity     float64 `json:"velocity"`
	Acceleration float64 `json:"acceleration"`

	Trend        string  `json:"trend"`         // UP / DOWN / FLAT
	TrendStrength float64 `json:"trend_strength"`

	Signal       string  `json:"signal"`        // BUY / SELL / HOLD
}

// ==========================
// ENGINE
// ==========================
func ComputeAlpha(state *MarketState) AlphaSignals {

	history := state.History
	n := len(history)

	if n == 0 {
		return AlphaSignals{}
	}

	var velocity float64
	var acceleration float64
	trend := "FLAT"
	signal := "HOLD"
	var strength float64

	// ==========================
	// VELOCITY
	// ==========================
	if n >= 2 {
		velocity = history[n-1] - history[n-2]
	}

	// ==========================
	// ACCELERATION
	// ==========================
	if n >= 3 {
		v1 := history[n-2] - history[n-3]
		v2 := history[n-1] - history[n-2]
		acceleration = v2 - v1
	}

	// ==========================
	// TREND DETECTION
	// ==========================
	if n >= 3 {
		a := history[n-3]
		b := history[n-2]
		c := history[n-1]

		if a < b && b < c {
			trend = "UP"
		} else if a > b && b > c {
			trend = "DOWN"
		}
	}

	// ==========================
	// TREND STRENGTH
	// ==========================
	strength = math.Abs(velocity) + math.Abs(acceleration)

	// ==========================
	// SIGNAL LOGIC (CORE EDGE)
	// ==========================

	// 🔥 STRONG BUY
	if trend == "UP" && velocity > 0 && acceleration > 0 {
		signal = "BUY"
	}

	// 🔥 STRONG SELL
	if trend == "DOWN" && velocity < 0 && acceleration < 0 {
		signal = "SELL"
	}

	// ⚠️ REVERSAL DETECTION
	if velocity > 0 && acceleration < 0 {
		signal = "WEAK_BUY"
	}

	if velocity < 0 && acceleration > 0 {
		signal = "WEAK_SELL"
	}

	// ==========================
	// NOISE FILTER (IMPORTANT)
	// ==========================
	if math.Abs(velocity) < 0.5 && math.Abs(acceleration) < 0.5 {
		signal = "HOLD"
		trend = "FLAT"
	}

	return AlphaSignals{
		LTPSequence: history,

		Velocity:     velocity,
		Acceleration: acceleration,

		Trend:         trend,
		TrendStrength: strength,

		Signal: signal,
	}
}


// ==========================
// LTP ALPHA (FOR ltpSeries)
// ==========================
func ComputeAlphaFromLTP(history []float64) AlphaSignals {

	n := len(history)

	var velocity float64
	var acceleration float64
	trend := "FLAT"
	signal := "HOLD"

	if n >= 2 {
		velocity = history[n-1] - history[n-2]
	}

	if n >= 3 {
		v1 := history[n-2] - history[n-3]
		v2 := history[n-1] - history[n-2]
		acceleration = v2 - v1

		a := history[n-3]
		b := history[n-2]
		c := history[n-1]

		if a < b && b < c {
			trend = "UP"
		} else if a > b && b > c {
			trend = "DOWN"
		}
	}

	// signals
	if trend == "UP" && velocity > 0 && acceleration > 0 {
		signal = "BUY"
	}

	if trend == "DOWN" && velocity < 0 && acceleration < 0 {
		signal = "SELL"
	}

	return AlphaSignals{
		LTPSequence: history,
		Velocity:    velocity,
		Trend:       trend,
		Signal:      signal,
	}
}