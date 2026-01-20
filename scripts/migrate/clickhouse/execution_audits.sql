-- Create execution_audits table in ClickHouse
-- Optimized for high-throughput trade logging and analytical queries

CREATE TABLE IF NOT EXISTS execution_audits (
    id String,
    trade_id String,
    order_id String,
    user_id String,
    symbol String,
    side String,
    price Float64,
    quantity Float64,
    fee Float64,
    venue String,
    algo_type String,
    timestamp Int64
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(toDateTime(timestamp / 1000))
ORDER BY (timestamp, symbol, user_id)
SETTINGS index_granularity = 8192;
