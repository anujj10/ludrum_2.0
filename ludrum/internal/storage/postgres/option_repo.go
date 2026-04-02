package postgres

import (
	"context"
	"log"
	"time"

	"ludrum/internal/storage/types"

	"github.com/jackc/pgx/v5"
)

// SaveOptionsBatch inserts multiple option rows efficiently
func SaveOptionsBatch(options []types.DBOption) {

	if len(options) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	batch := &pgx.Batch{}

	query := `
		INSERT INTO option_chain (
			time, symbol, strike, option_type,
			ltp, bid, ask,
			oi, oi_change, oi_change_pct,
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

		// log.Println("Batch size:", len(options))
	}

	br := DB.SendBatch(ctx, batch)

	// Execute all queries
	err := br.Close()
	if err != nil {
		log.Printf(" Batch insert failed: %v", err)
		return
	}

	log.Printf("Inserted %d option rows", len(options))
}