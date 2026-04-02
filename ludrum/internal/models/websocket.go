package models

type MarketTick struct {
    Symbol        string  `json:"symbol"`
    LTP           float64 `json:"ltp"`
    PrevClose     float64 `json:"prev_close_price"`
    Open          float64 `json:"open_price"`
    High          float64 `json:"high_price"`
    Low           float64 `json:"low_price"`
    Change        float64 `json:"ch"`
    ChangePercent float64 `json:"chp"`
    Timestamp     int64   `json:"exch_feed_time"`
}