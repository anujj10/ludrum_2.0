package processor

import (
	"math"
	"sort"

	"ludrum/internal/models"
)

// ==========================
// OUTPUT STRUCTS
// ==========================
type OptionLeg struct {
	LTP      float64 `json:"ltp"`
	OI       int64   `json:"oi"`
	OIChange int64   `json:"oi_change"`
	Volume   int64   `json:"volume"`
}

type StrikeView struct {
	Strike float64    `json:"strike"`
	CE     *OptionLeg `json:"ce,omitempty"`
	PE     *OptionLeg `json:"pe,omitempty"`
}

type OptionsChainPayload struct {
	Symbol  string        `json:"symbol"`
	Spot    float64       `json:"spot"`
	ATM     float64       `json:"atm"`
	Strikes []StrikeView  `json:"strikes"`
}

// ==========================
// 🔥 MAIN BUILDER
// ==========================
func BuildOptionsChainPayload(
	snap *models.MarketSnapshot,
	rangeSize float64,
) OptionsChainPayload {

	// 👉 ATM (adjust step if needed)
	atm := math.Round(snap.SpotPrice/50) * 50

	var strikes []StrikeView

	for strike, data := range snap.Strikes {

		// filter near ATM
		if math.Abs(strike-atm) > rangeSize {
			continue
		}

		view := StrikeView{
			Strike: strike,
		}

		if data.CE != nil {
			view.CE = &OptionLeg{
				LTP:      data.CE.LTP,
				OI:       data.CE.OI,
				OIChange: data.CE.OICh,
				Volume:   data.CE.Volume,
			}
		}

		if data.PE != nil {
			view.PE = &OptionLeg{
				LTP:      data.PE.LTP,
				OI:       data.PE.OI,
				OIChange: data.PE.OICh,
				Volume:   data.PE.Volume,
			}
		}

		strikes = append(strikes, view)
	}

	// ✅ sort (VERY IMPORTANT)
	sort.Slice(strikes, func(i, j int) bool {
		return strikes[i].Strike < strikes[j].Strike
	})

	return OptionsChainPayload{
		Symbol:  snap.Symbol,
		Spot:    snap.SpotPrice,
		ATM:     atm,
		Strikes: strikes,
	}
}