package processor

import (
	"ludrum/internal/models"
	"time"
)

func BuildMarketSnapshot(resp *models.OptionChainResponse) *models.MarketSnapshot {

	strikeMap := BuilderStrikeMap(resp.Data.OptionsChain)

	// Extract spot price (index row usually first element)
	var spot float64
	if len(resp.Data.OptionsChain) > 0 {
		spot = resp.Data.OptionsChain[0].LTP
	}

	// Extract VIX
	vix := resp.Data.IndiaVIX.LTP

	// Extract expiries (only dates for now)
	expiries := make([]string, 0, len(resp.Data.ExpiryData))
	for _, exp := range resp.Data.ExpiryData {
		expiries = append(expiries, exp.Date)
	}

	// Build final snapshot
	snapshot := &models.MarketSnapshot{
		Symbol:      "NIFTY",
		SpotPrice:   spot,
		VIX:         vix,
		Strikes:     strikeMap,
		TotalCallOI: resp.Data.CallOI,
		TotalPutOI:  resp.Data.PutOI,
		Timestamp:   time.Now().Unix(),
	}

	return snapshot
}