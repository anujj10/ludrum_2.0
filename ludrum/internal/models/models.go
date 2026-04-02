package models

type MarketInfo struct {
	Symbol        string  `json:"symbol"`
	LTP           float64 `json:"ltp"`
	Change        float64 `json:"change"`
	ChangePercent float64 `json:"change_percent"`
	Bias          string  `json:"bias"`
	Strength      int     `json:"strength"`
}

type MetaInfo struct {
	Timestamp int64  `json:"timestamp"`
	Source    string `json:"source"` // "ws" or "rest"
}

type Payload struct {
	Market  MarketInfo            `json:"market"`
	Signals map[string]interface{} `json:"signals"`
	Meta    MetaInfo             `json:"meta"`
}