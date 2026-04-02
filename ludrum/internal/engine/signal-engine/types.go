package signalengine

type SignalType string

const (
	StrongBullish SignalType = "STRONG_BULLISH"
	StrongBearish SignalType = "STRONG_BEARISH"
	WeakBullish   SignalType = "WEAK_BULLISH"
	WeakBearish   SignalType = "WEAK_BEARISH"
	Neutral       SignalType = "NEUTRAL"
)

type SmartSignal struct {
	Type       SignalType
	Strength   float64
	Confidence float64
}