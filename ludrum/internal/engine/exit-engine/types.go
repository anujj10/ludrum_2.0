package exitengine

type ExitType string

const (
	TrailingExit ExitType = "TRAILING_EXIT"
	MomentumExit ExitType = "MOMENTUM_FADE"
	TargetHit    ExitType = "TARGET_HIT"
	StopLossHit  ExitType = "STOP_LOSS"
	Hold         ExitType = "HOLD"
)

type ExitDecision struct {
	Exit   bool     `json:"exit"`
	Type   ExitType `json:"type"`
	Reason string   `json:"reason"`
}