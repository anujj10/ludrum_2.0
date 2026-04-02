package processor

import (
	"time"

	"ludrum/internal/models"
)

// 🔥 Convert tick → MarketSnapshot (compatible with pipeline)
func BuildMarketSnapshotFromTick(t models.MarketTick) *models.MarketSnapshot {

	return &models.MarketSnapshot{
		Timestamp: time.Now().Unix(),

		// minimal required fields
		SpotPrice: t.LTP,

		// optional (depends on your struct)
		// Underlying: t.Symbol,

		// you can extend later
	}
}