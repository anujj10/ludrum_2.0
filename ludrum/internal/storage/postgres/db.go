package postgres

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func buildDSN() string {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "ludrum")

	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		user, password, host, port, dbname,
	)
}

func getEnv(key, fallback string) string {
	val := os.Getenv(key)
	if val == "" {
		return fallback
	}
	return val
}

func InitDB() {
	dsn := buildDSN()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("Unable to create DB cool: %v", err)
	}

	err = pool.Ping(ctx)
	if err != nil {
		log.Fatalf("unable to connect to DB: %v", err)
	}

	DB = pool

	ensureTimescaleTables()

	fmt.Println("Connected to PostgreSQL (TimescaleDB)")
}

func ensureTimescaleTables() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := migrateLegacyMarketSnapshots(ctx); err != nil {
		log.Printf("failed to migrate legacy market_snapshots: %v", err)
	}

	statements := []string{
		`CREATE EXTENSION IF NOT EXISTS timescaledb`,
		`CREATE TABLE IF NOT EXISTS market_snapshots (
			time TIMESTAMPTZ NOT NULL,
			symbol TEXT,
			strike DOUBLE PRECISION,
			option_type TEXT,
			ltp DOUBLE PRECISION,
			bid DOUBLE PRECISION,
			ask DOUBLE PRECISION,
			oi BIGINT,
			oi_change BIGINT,
			oi_change_pct DOUBLE PRECISION,
			volume BIGINT
		)`,
		`SELECT create_hypertable('market_snapshots', 'time', if_not_exists => TRUE)`,
		`CREATE TABLE IF NOT EXISTS option_chain (
			time TIMESTAMPTZ NOT NULL,
			symbol TEXT,
			strike DOUBLE PRECISION,
			option_type TEXT,
			ltp DOUBLE PRECISION,
			bid DOUBLE PRECISION,
			ask DOUBLE PRECISION,
			oi BIGINT,
			oi_change BIGINT,
			oi_change_pct DOUBLE PRECISION,
			volume BIGINT
		)`,
		`SELECT create_hypertable('option_chain', 'time', if_not_exists => TRUE)`,
		`CREATE TABLE IF NOT EXISTS option_features (
			time TIMESTAMPTZ,
			symbol TEXT,
			strike DOUBLE PRECISION,
			option_type TEXT,
			oi BIGINT,
			oi_change BIGINT,
			volume BIGINT,
			oi_velocity DOUBLE PRECISION,
			volume_spike DOUBLE PRECISION,
			price_change DOUBLE PRECISION,
			bid_ask_spread DOUBLE PRECISION,
			spot_price DOUBLE PRECISION,
			distance_from_atm DOUBLE PRECISION
		)`,
		`SELECT create_hypertable('option_features', 'time', if_not_exists => TRUE)`,
		`CREATE TABLE IF NOT EXISTS option_oi_change_events (
			time TIMESTAMPTZ NOT NULL,
			symbol TEXT,
			strike DOUBLE PRECISION,
			option_type TEXT,
			oi_change BIGINT,
			ltp_change DOUBLE PRECISION
		)`,
		`SELECT create_hypertable('option_oi_change_events', 'time', if_not_exists => TRUE)`,
		`CREATE TABLE IF NOT EXISTS beta_users (
			id BIGSERIAL PRIMARY KEY,
			full_name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			phone TEXT NOT NULL UNIQUE,
			trading_style TEXT,
			client_id TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_credential_sent_at TIMESTAMPTZ
		)`,
		`CREATE TABLE IF NOT EXISTS email_otp_codes (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES beta_users(id) ON DELETE CASCADE,
			code_hash TEXT NOT NULL,
			expires_at TIMESTAMPTZ NOT NULL,
			consumed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_otp_codes_user_id_created_at ON email_otp_codes(user_id, created_at DESC)`,
		`CREATE TABLE IF NOT EXISTS auth_sessions (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES beta_users(id) ON DELETE CASCADE,
			token_hash TEXT NOT NULL UNIQUE,
			expires_at TIMESTAMPTZ NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_seen_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_auth_sessions_expires_at ON auth_sessions(expires_at)`,
		`CREATE TABLE IF NOT EXISTS fyers_accounts (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES beta_users(id) ON DELETE CASCADE,
			broker_user_id TEXT,
			app_id TEXT NOT NULL,
			redirect_uri TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'pending',
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			last_connected_at TIMESTAMPTZ,
			UNIQUE (user_id, app_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fyers_accounts_user_id ON fyers_accounts(user_id)`,
		`CREATE TABLE IF NOT EXISTS fyers_tokens (
			id BIGSERIAL PRIMARY KEY,
			account_id BIGINT NOT NULL UNIQUE REFERENCES fyers_accounts(id) ON DELETE CASCADE,
			access_token_encrypted TEXT NOT NULL,
			refresh_token_encrypted TEXT,
			token_type TEXT NOT NULL DEFAULT 'Bearer',
			expires_at TIMESTAMPTZ,
			refreshed_at TIMESTAMPTZ,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fyers_tokens_expires_at ON fyers_tokens(expires_at)`,
		`CREATE TABLE IF NOT EXISTS user_runtime_status (
			id BIGSERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES beta_users(id) ON DELETE CASCADE,
			account_id BIGINT NOT NULL REFERENCES fyers_accounts(id) ON DELETE CASCADE,
			runtime_state TEXT NOT NULL DEFAULT 'pending',
			last_ws_connect_at TIMESTAMPTZ,
			last_tick_at TIMESTAMPTZ,
			last_error TEXT,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, account_id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_user_runtime_status_account_id ON user_runtime_status(account_id)`,
		`CREATE TABLE IF NOT EXISTS user_runtime_snapshots (
			time TIMESTAMPTZ NOT NULL,
			user_id BIGINT NOT NULL REFERENCES beta_users(id) ON DELETE CASCADE,
			account_id BIGINT NOT NULL REFERENCES fyers_accounts(id) ON DELETE CASCADE,
			payload_type TEXT NOT NULL DEFAULT 'snapshot',
			spot DOUBLE PRECISION,
			pair_count INTEGER NOT NULL DEFAULT 0,
			payload_json JSONB NOT NULL
		)`,
		`SELECT create_hypertable('user_runtime_snapshots', 'time', if_not_exists => TRUE)`,
		`CREATE INDEX IF NOT EXISTS idx_user_runtime_snapshots_user_account_time ON user_runtime_snapshots(user_id, account_id, time DESC)`,
		`CREATE TABLE IF NOT EXISTS user_option_oi_change_events (
			time TIMESTAMPTZ NOT NULL,
			user_id BIGINT NOT NULL REFERENCES beta_users(id) ON DELETE CASCADE,
			account_id BIGINT NOT NULL REFERENCES fyers_accounts(id) ON DELETE CASCADE,
			symbol TEXT,
			strike DOUBLE PRECISION,
			option_type TEXT,
			oi_change BIGINT,
			ltp_change DOUBLE PRECISION
		)`,
		`SELECT create_hypertable('user_option_oi_change_events', 'time', if_not_exists => TRUE)`,
		`CREATE INDEX IF NOT EXISTS idx_user_option_oi_events_scope_time ON user_option_oi_change_events(user_id, account_id, symbol, strike, option_type, time DESC)`,
		`CREATE TABLE IF NOT EXISTS app_settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
	}

	for _, statement := range statements {
		if _, err := DB.Exec(ctx, statement); err != nil {
			log.Printf("failed to ensure schema object: %v", err)
		}
	}
}

func migrateLegacyMarketSnapshots(ctx context.Context) error {
	const legacyCheck = `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			  AND table_name = 'market_snapshots'
			  AND column_name = 'spot_price'
		)
	`

	var isLegacy bool
	if err := DB.QueryRow(ctx, legacyCheck).Scan(&isLegacy); err != nil {
		return err
	}

	if !isLegacy {
		return nil
	}

	if _, err := DB.Exec(ctx, `DROP TABLE IF EXISTS market_snapshots_legacy`); err != nil {
		return err
	}

	if _, err := DB.Exec(ctx, `ALTER TABLE market_snapshots RENAME TO market_snapshots_legacy`); err != nil {
		return err
	}

	return nil
}
