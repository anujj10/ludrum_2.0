package replay

import (
	"context"
	// "database/sql"
	"log"
	"sort"
	"time"

	"ludrum/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type OptionRow struct {
	Time       time.Time
	Symbol     string
	Strike     float64
	Type       string

	LTP    float64
	Bid    float64
	Ask    float64
	OI     int64
	Volume int64
}

type ReplayEngine struct {
	rows   []OptionRow
	index  int
	speed  time.Duration
	lastTs time.Time
}

func NewReplayEngine(db *pgxpool.Pool, symbol string, start, end time.Time, speed time.Duration) *ReplayEngine {
	rows := loadOptionData(db, symbol, start, end)

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Time.Before(rows[j].Time)
	})

	return &ReplayEngine{
		rows:  rows,
		index: 0,
		speed: speed,
	}
}

// ================= LOAD DATA =================

func loadOptionData(db *pgxpool.Pool, symbol string, start, end time.Time) []OptionRow {
	query := `
	SELECT time, symbol, strike, option_type, ltp, bid, ask, oi, volume
	FROM option_chain
	WHERE symbol = $1
	AND time BETWEEN $2 AND $3
	ORDER BY time ASC
	`

	rows, err := db.Query(context.Background(), query, symbol, start, end)
	if err != nil {
		log.Fatal("Replay DB error:", err)
	}
	defer rows.Close()

	var result []OptionRow

	for rows.Next() {
		var r OptionRow

		err := rows.Scan(
			&r.Time,
			&r.Symbol,
			&r.Strike,
			&r.Type,
			&r.LTP,
			&r.Bid,
			&r.Ask,
			&r.OI,
			&r.Volume,
		)
		if err != nil {
			log.Println("Scan error:", err)
			continue
		}

		result = append(result, r)
	}

	log.Printf("Replay loaded %d rows\n", len(result))

	return result
}

// ================= SNAPSHOT BUILDER =================

func (r *ReplayEngine) Next() *models.MarketSnapshot {
	if r.index >= len(r.rows) {
		return nil
	}

	currentTime := r.rows[r.index].Time

	snapshot := &models.MarketSnapshot{
		Symbol:    r.rows[r.index].Symbol,
		Strikes:   make(map[float64]*models.StrikeData),
		Timestamp: currentTime.Unix(),
	}

	for r.index < len(r.rows) && r.rows[r.index].Time.Equal(currentTime) {
		row := r.rows[r.index]

		if _, ok := snapshot.Strikes[row.Strike]; !ok {
			snapshot.Strikes[row.Strike] = &models.StrikeData{
				Strike: row.Strike,
			}
		}

		strikeData := snapshot.Strikes[row.Strike]

		option := &models.OptionChain{
			Ask:         row.Ask,
			Bid:         row.Bid,
			LTP:         row.LTP,
			OI:          row.OI,
			Volume:      row.Volume,
			OptionType:  row.Type,
			StrikePrice: row.Strike,
			Symbol:      row.Symbol,
		}

		if row.Type == "CE" {
			strikeData.CE = option
		} else if row.Type == "PE" {
			strikeData.PE = option
		}

		r.index++
	}

	// crude spot approximation (ATM strike)
	snapshot.SpotPrice = estimateSpot(snapshot)

	return snapshot
}

// ================= SPOT ESTIMATION =================

func estimateSpot(snapshot *models.MarketSnapshot) float64 {
	var closest float64
	minDiff := 1e9

	for strike := range snapshot.Strikes {
		diff := abs(strike - snapshot.Strikes[strike].Strike)

		if diff < minDiff {
			minDiff = diff
			closest = strike
		}
	}

	return closest
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// ================= RUN LOOP =================

func (r *ReplayEngine) Start(ctx context.Context, handler func(*models.MarketSnapshot)) {
	ticker := time.NewTicker(r.speed)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Replay stopped")
			return

		case <-ticker.C:
			snap := r.Next()
			if snap == nil {
				log.Println("Replay finished")
				return
			}

			handler(snap)
		}
	}
}