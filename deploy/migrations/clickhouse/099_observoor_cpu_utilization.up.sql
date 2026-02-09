-- Migration 099: add observoor CPU utilization summary table.

CREATE TABLE observoor.cpu_utilization_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    total_on_cpu_ns Float32 CODEC(ZSTD(1)),
    event_count UInt32 CODEC(ZSTD(1)),
    active_cores UInt16 CODEC(ZSTD(1)),
    system_cores UInt16 CODEC(ZSTD(1)),
    max_core_on_cpu_ns Float32 CODEC(ZSTD(1)),
    max_core_id UInt32 CODEC(ZSTD(1)),
    mean_core_pct Float32 CODEC(ZSTD(1)),
    min_core_pct Float32 CODEC(ZSTD(1)),
    max_core_pct Float32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.cpu_utilization ON CLUSTER '{cluster}' AS observoor.cpu_utilization_local
ENGINE = Distributed(
    '{cluster}',
    'observoor',
    cpu_utilization_local,
    cityHash64(window_start, meta_network_name, meta_client_name)
);
