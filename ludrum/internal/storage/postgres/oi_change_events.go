package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"ludrum/internal/storage/types"

	"github.com/jackc/pgx/v5"
)

type optionSnapshotState struct {
	OI  int64
	LTP float64
}

type optionEventState struct {
	LTP float64
}

var (
	optionWriteMu     sync.Mutex
	lastSnapshotByKey = map[string]optionSnapshotState{}
	lastEventByKey    = map[string]optionEventState{}
)

func optionStateKey(symbol string, strike float64, optionType string) string {
	return fmt.Sprintf("%s|%.2f|%s", symbol, strike, optionType)
}

func loadLastSnapshotState(ctx context.Context, opt types.DBOption) (optionSnapshotState, bool, error) {
	row := DB.QueryRow(
		ctx,
		`SELECT oi, ltp
		 FROM market_snapshots
		 WHERE symbol = $1 AND strike = $2 AND option_type = $3
		 ORDER BY time DESC
		 LIMIT 1`,
		opt.Symbol,
		opt.Strike,
		opt.OptionType,
	)

	var state optionSnapshotState
	if err := row.Scan(&state.OI, &state.LTP); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return optionSnapshotState{}, false, nil
		}
		return optionSnapshotState{}, false, err
	}

	return state, true, nil
}

func saveOIChangeEvents(ctx context.Context, events []types.DBOIChangeEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `
		INSERT INTO option_oi_change_events (
			time, symbol, strike, option_type, oi_change, ltp_change
		)
		VALUES ($1,$2,$3,$4,$5,$6)
	`

	for _, event := range events {
		batch.Queue(
			query,
			event.Time,
			event.Symbol,
			event.Strike,
			event.OptionType,
			event.OIChange,
			event.LTPChange,
		)
	}

	br := DB.SendBatch(ctx, batch)
	return br.Close()
}

func SaveOptionsAndOIEvents(options []types.DBOption) {
	if len(options) == 0 {
		return
	}

	optionWriteMu.Lock()
	defer optionWriteMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	events := make([]types.DBOIChangeEvent, 0, len(options))

	for _, opt := range options {
		key := optionStateKey(opt.Symbol, opt.Strike, opt.OptionType)

		snapshotState, hasSnapshot := lastSnapshotByKey[key]
		if !hasSnapshot {
			loaded, found, err := loadLastSnapshotState(ctx, opt)
			if err != nil {
				log.Printf("failed to load previous option snapshot for %.0f %s: %v", opt.Strike, opt.OptionType, err)
			} else if found {
				snapshotState = loaded
				hasSnapshot = true
				lastSnapshotByKey[key] = loaded
			}
		}

		if hasSnapshot {
			oiDelta := opt.OI - snapshotState.OI
			if oiDelta != 0 {
				eventState, hasEvent := lastEventByKey[key]
				if !hasEvent {
					loaded, found, err := loadLastEventLTP(ctx, opt)
					if err != nil {
						log.Printf("failed to load previous OI event for %.0f %s: %v", opt.Strike, opt.OptionType, err)
					} else if found {
						eventState = loaded
						hasEvent = true
						lastEventByKey[key] = loaded
					}
				}

				ltpDelta := 0.0
				if hasEvent {
					ltpDelta = opt.LTP - eventState.LTP
				} else {
					ltpDelta = opt.LTP - snapshotState.LTP
				}

				events = append(events, types.DBOIChangeEvent{
					Time:       opt.Time,
					Symbol:     opt.Symbol,
					Strike:     opt.Strike,
					OptionType: opt.OptionType,
					OIChange:   oiDelta,
					LTPChange:  ltpDelta,
				})

				lastEventByKey[key] = optionEventState{LTP: opt.LTP}
			}
		}

		lastSnapshotByKey[key] = optionSnapshotState{
			OI:  opt.OI,
			LTP: opt.LTP,
		}
	}

	if err := saveOIChangeEvents(ctx, events); err != nil {
		log.Printf("failed to insert OI change events: %v", err)
		return
	}

	if len(events) > 0 {
		log.Printf("Inserted %d OI change event rows", len(events))
	}
}

func loadLastEventLTP(ctx context.Context, opt types.DBOption) (optionEventState, bool, error) {
	row := DB.QueryRow(
		ctx,
		`SELECT chain.ltp
		 FROM option_oi_change_events events
		 JOIN LATERAL (
			SELECT ltp
			FROM market_snapshots
			WHERE symbol = events.symbol AND strike = events.strike AND option_type = events.option_type
			AND time <= events.time
			ORDER BY time DESC
			LIMIT 1
		 ) chain ON true
		 WHERE events.symbol = $1 AND events.strike = $2 AND events.option_type = $3
		 ORDER BY events.time DESC
		 LIMIT 1`,
		opt.Symbol,
		opt.Strike,
		opt.OptionType,
	)

	var ltp float64
	if err := row.Scan(&ltp); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return optionEventState{}, false, nil
		}
		return optionEventState{}, false, err
	}

	return optionEventState{LTP: ltp}, true, nil
}
