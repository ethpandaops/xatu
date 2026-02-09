-- Migration 005: Replace local_port UInt16 with port_label LowCardinality(String)
-- Drops and recreates 4 tables (net_io, tcp_retransmit, tcp_rtt, tcp_cwnd)
-- plus their distributed counterparts. Historical data in these tables is lost.
-- local_port is in ORDER BY so ALTER TABLE cannot change the column type.

--------------------------------------------------------------------------------
-- COUNTER TABLES - NETWORK (net_io, tcp_retransmit)
-- Replace local_port UInt16 with port_label LowCardinality(String)
--------------------------------------------------------------------------------

-- net_io: drop distributed first, then local
DROP TABLE IF EXISTS observoor.net_io ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.net_io_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE observoor.net_io_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    port_label LowCardinality(String),
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
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label, direction);

CREATE TABLE observoor.net_io ON CLUSTER '{cluster}' AS observoor.net_io_local
ENGINE = Distributed('{cluster}', 'observoor', net_io_local, cityHash64(window_start, meta_network_name, meta_client_name));

-- tcp_retransmit: drop distributed first, then local
DROP TABLE IF EXISTS observoor.tcp_retransmit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_retransmit_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE observoor.tcp_retransmit_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    port_label LowCardinality(String),
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
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label, direction);

CREATE TABLE observoor.tcp_retransmit ON CLUSTER '{cluster}' AS observoor.tcp_retransmit_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_retransmit_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- GAUGE TABLES - TCP (tcp_rtt, tcp_cwnd)
-- Replace local_port UInt16 with port_label LowCardinality(String)
--------------------------------------------------------------------------------

-- tcp_rtt: drop distributed first, then local
DROP TABLE IF EXISTS observoor.tcp_rtt ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_rtt_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE observoor.tcp_rtt_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    port_label LowCardinality(String),
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
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label);

CREATE TABLE observoor.tcp_rtt ON CLUSTER '{cluster}' AS observoor.tcp_rtt_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_rtt_local, cityHash64(window_start, meta_network_name, meta_client_name));

-- tcp_cwnd: drop distributed first, then local
DROP TABLE IF EXISTS observoor.tcp_cwnd ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_cwnd_local ON CLUSTER '{cluster}' SYNC;

CREATE TABLE observoor.tcp_cwnd_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    port_label LowCardinality(String),
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
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label);

CREATE TABLE observoor.tcp_cwnd ON CLUSTER '{cluster}' AS observoor.tcp_cwnd_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_cwnd_local, cityHash64(window_start, meta_network_name, meta_client_name));
