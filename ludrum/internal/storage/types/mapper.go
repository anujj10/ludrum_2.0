package types

import (
	"time"
	"ludrum/internal/models"
)

// Convert snapshot → DB snapshot
func ToDBSnapshot(snap *models.MarketSnapshot) DBSnapshot {
	return DBSnapshot{
		Time:        time.Unix(snap.Timestamp, 0),
		Symbol:      snap.Symbol,
		SpotPrice:   snap.SpotPrice,
		VIX:         snap.VIX,
		TotalCallOI: snap.TotalCallOI,
		TotalPutOI:  snap.TotalPutOI,
	}
}

// Convert full snapshot → flattened option rows
func ToDBOptions(snap *models.MarketSnapshot) []DBOption {

	var result []DBOption
	t := time.Unix(snap.Timestamp, 0)

	for strike, data := range snap.Strikes {

		if data.CE != nil {
			result = append(result, DBOption{
				Time:       t,
				Symbol:     snap.Symbol,
				Strike:     strike,
				OptionType: "CE",

				LTP: data.CE.LTP,
				Bid: data.CE.Bid,
				Ask: data.CE.Ask,

				OI:          data.CE.OI,
				OIChange:    data.CE.OICh,
				OIChangePct: data.CE.OIChp,
				Volume:      data.CE.Volume,
			})
		}

		if data.PE != nil {
			result = append(result, DBOption{
				Time:       t,
				Symbol:     snap.Symbol,
				Strike:     strike,
				OptionType: "PE",

				LTP: data.PE.LTP,
				Bid: data.PE.Bid,
				Ask: data.PE.Ask,

				OI:          data.PE.OI,
				OIChange:    data.PE.OICh,
				OIChangePct: data.PE.OIChp,
				Volume:      data.PE.Volume,
			})
		}
	}

	return result
}
