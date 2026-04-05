package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"ludrum/internal/models"
	"ludrum/internal/storage/types"

	"github.com/jackc/pgx/v5"
)

func SaveUserRuntimeSnapshot(ctx context.Context, userID, accountID int64, payload models.StreamPayload) error {
	if userID == 0 || accountID == 0 {
		return nil
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	_, err = DB.Exec(
		writeCtx,
		`
		INSERT INTO user_runtime_snapshots (
			time, user_id, account_id, payload_type, spot, pair_count, payload_json
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		`,
		time.Now().UTC(),
		userID,
		accountID,
		payload.Type,
		payload.Spot,
		len(payload.Pairs),
		body,
	)
	return err
}

func LoadLatestUserRuntimeSnapshot(ctx context.Context, userID, accountID int64) (models.StreamPayload, bool, error) {
	if userID == 0 || accountID == 0 {
		return models.StreamPayload{}, false, nil
	}

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	row := DB.QueryRow(
		readCtx,
		`
		SELECT payload_json
		FROM user_runtime_snapshots
		WHERE user_id = $1 AND account_id = $2
		ORDER BY time DESC
		LIMIT 1
		`,
		userID,
		accountID,
	)

	var body []byte
	if err := row.Scan(&body); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.StreamPayload{}, false, nil
		}
		return models.StreamPayload{}, false, err
	}

	var payload models.StreamPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return models.StreamPayload{}, false, err
	}

	return payload, true, nil
}

func SaveUserRuntimeOIEvents(ctx context.Context, events []types.UserScopedOIChangeEvent) error {
	if len(events) == 0 {
		return nil
	}

	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	batch := &pgx.Batch{}
	query := `
		INSERT INTO user_option_oi_change_events (
			time, user_id, account_id, symbol, strike, option_type, oi_change, ltp_change
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
	`

	for _, event := range events {
		batch.Queue(
			query,
			event.Time,
			event.UserID,
			event.AccountID,
			event.Symbol,
			event.Strike,
			event.OptionType,
			event.OIChange,
			event.LTPChange,
		)
	}

	return DB.SendBatch(writeCtx, batch).Close()
}

func LoadUserRuntimeOIEvents(ctx context.Context, userID, accountID int64, symbol string, strikes []float64, limit int) ([]types.DBOIChangeEvent, error) {
	if userID == 0 || accountID == 0 || len(strikes) == 0 {
		return nil, nil
	}

	if limit <= 0 {
		limit = 12
	}

	readCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	rows, err := DB.Query(
		readCtx,
		`
		WITH ranked AS (
			SELECT
				time,
				symbol,
				strike,
				option_type,
				oi_change,
				ltp_change,
				ROW_NUMBER() OVER (
					PARTITION BY strike, option_type
					ORDER BY time DESC
				) AS rn
			FROM user_option_oi_change_events
			WHERE user_id = $1
			  AND account_id = $2
			  AND symbol = $3
			  AND strike = ANY($4)
		)
		SELECT time, symbol, strike, option_type, oi_change, ltp_change
		FROM ranked
		WHERE rn <= $5
		ORDER BY strike ASC, option_type ASC, time DESC
		`,
		userID,
		accountID,
		symbol,
		strikes,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]types.DBOIChangeEvent, 0)
	for rows.Next() {
		var row types.DBOIChangeEvent
		if err := rows.Scan(&row.Time, &row.Symbol, &row.Strike, &row.OptionType, &row.OIChange, &row.LTPChange); err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}
