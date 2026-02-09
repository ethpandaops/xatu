-- Migration 005 rollback: Restore local_port UInt16 from port_label LowCardinality(String)
-- Drops and recreates 4 tables (net_io, tcp_retransmit, tcp_rtt, tcp_cwnd)
-- plus their distributed counterparts. Data is lost on rollback.

--------------------------------------------------------------------------------
-- COUNTER TABLES - NETWORK (net_io, tcp_retransmit)
-- Restore local_port UInt16
--------------------------------------------------------------------------------

-- net_io
DROP TABLE IF EXISTS net_io ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS net_io_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE net_io_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    direction LowCardinality(String),
    sum Float32 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, local_port, direction);

CREATE TABLE net_io ON CLUSTER '{cluster}' AS net_io_local
ENGINE = Distributed('{cluster}', 'observoor', net_io_local, cityHash64(window_start, meta_network_name, meta_client_name));

-- tcp_retransmit
DROP TABLE IF EXISTS tcp_retransmit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tcp_retransmit_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE tcp_retransmit_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    direction LowCardinality(String),
    sum Float32 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, local_port, direction);

CREATE TABLE tcp_retransmit ON CLUSTER '{cluster}' AS tcp_retransmit_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_retransmit_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- GAUGE TABLES - TCP (tcp_rtt, tcp_cwnd)
-- Restore local_port UInt16
--------------------------------------------------------------------------------

-- tcp_rtt
DROP TABLE IF EXISTS tcp_rtt ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tcp_rtt_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE tcp_rtt_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    sum Float32 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Float32 CODEC(ZSTD(1)),
    max Float32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, local_port);

CREATE TABLE tcp_rtt ON CLUSTER '{cluster}' AS tcp_rtt_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_rtt_local, cityHash64(window_start, meta_network_name, meta_client_name));

-- tcp_cwnd
DROP TABLE IF EXISTS tcp_cwnd ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tcp_cwnd_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE tcp_cwnd_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    sum Float32 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Float32 CODEC(ZSTD(1)),
    max Float32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, local_port);

CREATE TABLE tcp_cwnd ON CLUSTER '{cluster}' AS tcp_cwnd_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_cwnd_local, cityHash64(window_start, meta_network_name, meta_client_name));
