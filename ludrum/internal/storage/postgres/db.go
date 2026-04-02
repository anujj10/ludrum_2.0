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
