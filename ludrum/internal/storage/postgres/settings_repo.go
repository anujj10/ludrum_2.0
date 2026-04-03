package postgres

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5"
)

const marketOverrideKey = "market_override_reason"

func GetMarketOverride(ctx context.Context) (string, error) {
	var value string
	err := DB.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, marketOverrideKey).Scan(&value)
	if err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(value), nil
}

func SetMarketOverride(ctx context.Context, reason string) error {
	reason = strings.TrimSpace(reason)
	_, err := DB.Exec(
		ctx,
		`
		INSERT INTO app_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key)
		DO UPDATE SET value = EXCLUDED.value, updated_at = NOW()
		`,
		marketOverrideKey,
		reason,
	)
	return err
}

func ClearMarketOverride(ctx context.Context) error {
	_, err := DB.Exec(ctx, `DELETE FROM app_settings WHERE key = $1`, marketOverrideKey)
	return err
}
