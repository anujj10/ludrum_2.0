package postgres

import (
	"context"
	"log"
	"time"

	"ludrum/internal/storage/types"

	"github.com/jackc/pgx/v5"
)

// SaveMinuteSnapshots stores option-row snapshots once per minute using
// the same column shape as option_chain.
func SaveMinuteSnapshots(options []types.DBOption) {
	if len(options) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	query := `
		INSERT INTO market_snapshots (
			time,
			symbol,
			strike,
			option_type,
			ltp,
			bid,
			ask,
			oi,
			oi_change,
			oi_change_pct,
			volume
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
	`

	for _, opt := range options {
		batch.Queue(
			query,
			opt.Time,
			opt.Symbol,
			opt.Strike,
			opt.OptionType,
			opt.LTP,
			opt.Bid,
			opt.Ask,
			opt.OI,
			opt.OIChange,
			opt.OIChangePct,
			opt.Volume,
		)
	}

	br := DB.SendBatch(ctx, batch)
	if err := br.Close(); err != nil {
		log.Printf("Failed to insert market snapshots: %v", err)
		return
	}

	log.Printf("Saved %d minute market snapshot rows", len(options))
}
