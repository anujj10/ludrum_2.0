package entryengine

type EntryType string

const (
	PullbackEntry EntryType = "PULLBACK"
	BreakoutEntry EntryType = "BREAKOUT"
	NoEntry       EntryType = "NO_ENTRY"
)

type EntryDecision struct {
	Valid   bool      `json:"valid"`
	Type    EntryType `json:"type"`
	Reason  string    `json:"reason"`
}