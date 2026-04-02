package models

type OptionChainResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Status  string `json:"s"`
	Data    Data   `json:"data"`
}

type Data struct {
	CallOI      int64          `json:"callOi"`
	PutOI       int64          `json:"putOi"`
	ExpiryData  []Expiry       `json:"expiryData"`
	IndiaVIX    Instrument     `json:"indiavixData"`
	OptionsChain []OptionChain `json:"optionsChain"`
}

type Expiry struct {
	Date   string `json:"date"`
	Expiry string `json:"expiry"` // keep string (API gives string timestamp)
}

type Instrument struct {
	Ask         float64 `json:"ask"`
	Bid         float64 `json:"bid"`
	Description string  `json:"description"`
	ExSymbol    string  `json:"ex_symbol"`
	Exchange    string  `json:"exchange"`
	FyToken     string  `json:"fyToken"`
	LTP         float64 `json:"ltp"`
	LTPCh       float64 `json:"ltpch"`
	LTPChp      float64 `json:"ltpchp"`
	OptionType  string  `json:"option_type"`
	StrikePrice float64 `json:"strike_price"`
	Symbol      string  `json:"symbol"`
	FP    float64 `json:"fp,omitempty"`
	FPCh  float64 `json:"fpch,omitempty"`
	FPChp float64 `json:"fpchp,omitempty"`
}

type OptionChain struct {
	Ask         float64 `json:"ask"`
	Bid         float64 `json:"bid"`
	FyToken     string  `json:"fyToken"`
	LTP         float64 `json:"ltp"`
	LTPCh       float64 `json:"ltpch"`
	LTPChp      float64 `json:"ltpchp"`
	OI          int64   `json:"oi"`
	OICh        int64   `json:"oich"`
	OIChp       float64 `json:"oichp"`
	PrevOI      int64   `json:"prev_oi"`
	OptionType  string  `json:"option_type"`
	StrikePrice float64 `json:"strike_price"`
	Symbol      string  `json:"symbol"`
	Volume      int64   `json:"volume"`
}

