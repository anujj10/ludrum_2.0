package postgres

import (
	"time"

	"ludrum/internal/models"
	"ludrum/internal/storage/types"
)

// Convert API response → DB format
func ConvertToDBOptions(options []models.OptionChain) []types.DBOption {

	var result []types.DBOption
	now := time.Now().UTC()

	for _, opt := range options {
		result = append(result, types.DBOption{
			Time:        now,
			Symbol:      "NIFTY",
			Strike:      opt.StrikePrice,
			OptionType:  opt.OptionType,
			LTP:         opt.LTP,
			Bid:         opt.Bid,
			Ask:         opt.Ask,
			OI:          opt.OI,
			OIChange:    opt.OICh,
			OIChangePct: opt.OIChp,
			Volume:      opt.Volume,
		})
	}

	return result
}
