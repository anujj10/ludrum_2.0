package payloadbuilder

type FinalPayload struct {
	Market    MarketState    `json:"market"`
	Signals   SignalBlock    `json:"signals"`
	Trade     TradeDecision  `json:"trade"`
	Risk      RiskBlock      `json:"risk"`
	Positions []Position     `json:"positions"`
}

type MarketState struct {
	Bias       string  `json:"bias"`
	Strength   int     `json:"strength"`
	Regime     string  `json:"regime"`
	Confidence float64 `json:"confidence"`
}

type SignalBlock struct {
	OIVelocity OIVelocity `json:"oi_velocity"`
	LTPSeq     []float64  `json:"ltp_sequence"`
}

type OIVelocity struct {
	CE float64 `json:"ce"`
	PE float64 `json:"pe"`
}

type RiskBlock struct {
	Capital      float64 `json:"capital"`
	RiskPerTrade float64 `json:"risk_per_trade"`
	PositionSize int     `json:"position_size"`
}

type Position struct {
	Symbol  string  `json:"symbol"`
	Strike  float64 `json:"strike"`
	Type    string  `json:"type"`
	Entry   float64 `json:"entry"`
	Current float64 `json:"current"`
	PnL     float64 `json:"pnl"`
} 


type TradeDecision struct {
	Action     string  `json:"action"`
	Symbol     string  `json:"symbol"`
	Strike     float64 `json:"strike"`
	OptionType string  `json:"option_type"`
	Entry      float64 `json:"entry"`
	SL         float64 `json:"sl"`
	Target     float64 `json:"target"`
	RR         float64 `json:"rr"`
	Score      float64  `json:"score"`
	Confidence float64  `json:"confidence"`
	Reasons    []string `json:"reasons"`
}