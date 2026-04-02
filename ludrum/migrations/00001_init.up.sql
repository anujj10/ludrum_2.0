-- Enable TimescaleDB
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- =========================
-- MARKET SNAPSHOTS
-- =========================
CREATE TABLE IF NOT EXISTS market_snapshots (
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
);

SELECT create_hypertable('market_snapshots', 'time', if_not_exists => TRUE);

-- =========================
-- OPTION CHAIN
-- =========================
CREATE TABLE IF NOT EXISTS option_chain (
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
);

SELECT create_hypertable('option_chain', 'time', if_not_exists => TRUE);

-- =========================
-- FEATURE TABLE (YOUR EDGE)
-- =========================
CREATE TABLE IF NOT EXISTS option_features (
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
);

SELECT create_hypertable('option_features', 'time', if_not_exists => TRUE);

-- =========================
-- OI CHANGE EVENTS
-- =========================
CREATE TABLE IF NOT EXISTS option_oi_change_events (
    time TIMESTAMPTZ NOT NULL,
    symbol TEXT,
    strike DOUBLE PRECISION,
    option_type TEXT,
    oi_change BIGINT,
    ltp_change DOUBLE PRECISION
);

SELECT create_hypertable('option_oi_change_events', 'time', if_not_exists => TRUE);
