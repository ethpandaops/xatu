-- observoor schema (database-agnostic).
-- Applied to the `observoor` database via the migration matrix.
-- Target database supplied at apply time do not hardcode it here.

CREATE TABLE IF NOT EXISTS block_merge_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `device_id` UInt32 CODEC(ZSTD(1)),
    `rw` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw)
COMMENT 'Aggregated block device I/O merge metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS cpu_utilization_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `total_on_cpu_ns` Float32 CODEC(ZSTD(1)),
    `event_count` UInt32 CODEC(ZSTD(1)),
    `active_cores` UInt16 CODEC(ZSTD(1)),
    `system_cores` UInt16 CODEC(ZSTD(1)),
    `max_core_on_cpu_ns` Float32 CODEC(ZSTD(1)),
    `max_core_id` UInt32 CODEC(ZSTD(1)),
    `mean_core_pct` Float32 CODEC(ZSTD(1)),
    `min_core_pct` Float32 CODEC(ZSTD(1)),
    `max_core_pct` Float32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated CPU utilization metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS disk_bytes_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `device_id` UInt32 CODEC(ZSTD(1)),
    `rw` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw)
COMMENT 'Aggregated disk I/O byte metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS disk_latency_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `device_id` UInt32 CODEC(ZSTD(1)),
    `rw` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw)
COMMENT 'Aggregated disk I/O latency metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS disk_queue_depth_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `device_id` UInt32 CODEC(ZSTD(1)),
    `rw` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw)
COMMENT 'Aggregated disk queue depth metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS fd_close_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated file descriptor close metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS fd_open_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated file descriptor open metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS host_specs_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `event_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `host_id` String,
    `kernel_release` LowCardinality(String),
    `os_name` LowCardinality(String),
    `architecture` LowCardinality(String),
    `cpu_model` String,
    `cpu_vendor` LowCardinality(String),
    `cpu_online_cores` UInt16 CODEC(ZSTD(1)),
    `cpu_logical_cores` UInt16 CODEC(ZSTD(1)),
    `cpu_physical_cores` UInt16 CODEC(ZSTD(1)),
    `cpu_performance_cores` UInt16 CODEC(ZSTD(1)),
    `cpu_efficiency_cores` UInt16 CODEC(ZSTD(1)),
    `cpu_unknown_type_cores` UInt16 CODEC(ZSTD(1)),
    `cpu_logical_ids` Array(UInt16),
    `cpu_core_ids` Array(Int32),
    `cpu_package_ids` Array(Int32),
    `cpu_die_ids` Array(Int32),
    `cpu_cluster_ids` Array(Int32),
    `cpu_core_types` Array(UInt8),
    `cpu_core_type_labels` Array(String),
    `cpu_online_flags` Array(UInt8),
    `cpu_max_freq_khz` Array(UInt64),
    `cpu_base_freq_khz` Array(UInt64),
    `memory_total_bytes` UInt64 CODEC(ZSTD(1)),
    `memory_type` LowCardinality(String),
    `memory_speed_mts` UInt32 CODEC(ZSTD(1)),
    `memory_dimm_count` UInt16 CODEC(ZSTD(1)),
    `memory_dimm_sizes_bytes` Array(UInt64),
    `memory_dimm_types` Array(String),
    `memory_dimm_speeds_mts` Array(UInt32),
    `memory_dimm_configured_speeds_mts` Array(UInt32),
    `memory_dimm_locators` Array(String),
    `memory_dimm_bank_locators` Array(String),
    `memory_dimm_manufacturers` Array(String),
    `memory_dimm_part_numbers` Array(String),
    `memory_dimm_serials` Array(String),
    `disk_count` UInt16 CODEC(ZSTD(1)),
    `disk_total_bytes` UInt64 CODEC(ZSTD(1)),
    `disk_names` Array(String),
    `disk_models` Array(String),
    `disk_vendors` Array(String),
    `disk_serials` Array(String),
    `disk_sizes_bytes` Array(UInt64),
    `disk_rotational` Array(UInt8),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(event_time))
ORDER BY (meta_network_name, event_time, host_id, meta_client_name)
COMMENT 'Periodic host hardware specification snapshots including CPU, memory, and disk details';

CREATE TABLE IF NOT EXISTS mem_compaction_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated memory compaction metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS mem_reclaim_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated memory reclaim metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS memory_usage_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String),
    `sampling_rate` Float32 CODEC(ZSTD(1)),
    `vm_size_bytes` UInt64 CODEC(ZSTD(1)),
    `vm_rss_bytes` UInt64 CODEC(ZSTD(1)),
    `rss_anon_bytes` UInt64 CODEC(ZSTD(1)),
    `rss_file_bytes` UInt64 CODEC(ZSTD(1)),
    `rss_shmem_bytes` UInt64 CODEC(ZSTD(1)),
    `vm_swap_bytes` UInt64 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Periodic memory usage snapshots of Ethereum client processes from /proc/[pid]/status';

CREATE TABLE IF NOT EXISTS net_io_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `port_label` LowCardinality(String),
    `direction` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label, direction)
COMMENT 'Aggregated network I/O metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS oom_kill_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated OOM kill events from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS page_fault_major_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated major page fault metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS page_fault_minor_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated minor page fault metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS process_exit_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated process exit events from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS process_fd_usage_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String),
    `sampling_rate` Float32 CODEC(ZSTD(1)),
    `open_fds` UInt32 CODEC(ZSTD(1)),
    `fd_limit_soft` UInt64 CODEC(ZSTD(1)),
    `fd_limit_hard` UInt64 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Periodic file descriptor usage snapshots of Ethereum client processes from /proc/[pid]/fd and /proc/[pid]/limits';

CREATE TABLE IF NOT EXISTS process_io_usage_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String),
    `sampling_rate` Float32 CODEC(ZSTD(1)),
    `rchar_bytes` UInt64 CODEC(ZSTD(1)),
    `wchar_bytes` UInt64 CODEC(ZSTD(1)),
    `syscr` UInt64 CODEC(ZSTD(1)),
    `syscw` UInt64 CODEC(ZSTD(1)),
    `read_bytes` UInt64 CODEC(ZSTD(1)),
    `write_bytes` UInt64 CODEC(ZSTD(1)),
    `cancelled_write_bytes` Int64 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Periodic I/O usage snapshots of Ethereum client processes from /proc/[pid]/io';

CREATE TABLE IF NOT EXISTS process_sched_usage_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String),
    `sampling_rate` Float32 CODEC(ZSTD(1)),
    `threads` UInt32 CODEC(ZSTD(1)),
    `voluntary_ctxt_switches` UInt64 CODEC(ZSTD(1)),
    `nonvoluntary_ctxt_switches` UInt64 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Periodic scheduler usage snapshots of Ethereum client processes from /proc/[pid]/status and /proc/[pid]/sched';

CREATE TABLE IF NOT EXISTS raw_events_local ON CLUSTER '{cluster}'
(
    `timestamp_ns` UInt64 COMMENT 'Wall clock time of the event in nanoseconds since Unix epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt64 COMMENT 'Ethereum slot number at the time of the event (from wall clock)' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) COMMENT 'Wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `cl_syncing` Bool COMMENT 'Whether the consensus layer was syncing when this event was captured' CODEC(ZSTD(1)),
    `el_optimistic` Bool COMMENT 'Whether the execution layer was in optimistic sync mode when this event was captured' CODEC(ZSTD(1)),
    `el_offline` Bool COMMENT 'Whether the execution layer was unreachable when this event was captured' CODEC(ZSTD(1)),
    `pid` UInt32 COMMENT 'Process ID of the traced Ethereum client' CODEC(ZSTD(1)),
    `tid` UInt32 COMMENT 'Thread ID within the traced process' CODEC(ZSTD(1)),
    `event_type` LowCardinality(String) COMMENT 'Type of eBPF event (syscall_read, disk_io, net_tx, etc.)',
    `client_type` LowCardinality(String) COMMENT 'Ethereum client implementation (geth, reth, prysm, lighthouse, etc.)',
    `latency_ns` UInt64 COMMENT 'Latency in nanoseconds for syscall and disk I/O events' CODEC(ZSTD(1)),
    `bytes` Int64 COMMENT 'Byte count for I/O events' CODEC(ZSTD(1)),
    `src_port` UInt16 COMMENT 'Source port for network events' CODEC(ZSTD(1)),
    `dst_port` UInt16 COMMENT 'Destination port for network events' CODEC(ZSTD(1)),
    `fd` Int32 COMMENT 'File descriptor number' CODEC(ZSTD(1)),
    `filename` String COMMENT 'Filename for fd_open events' CODEC(ZSTD(1)),
    `voluntary` Bool COMMENT 'Whether a context switch was voluntary' CODEC(ZSTD(1)),
    `on_cpu_ns` UInt64 COMMENT 'Time spent on CPU in nanoseconds before a context switch' CODEC(ZSTD(1)),
    `runqueue_ns` UInt64 COMMENT 'Time spent waiting in the run queue' CODEC(ZSTD(1)),
    `off_cpu_ns` UInt64 COMMENT 'Time spent off CPU' CODEC(ZSTD(1)),
    `major` Bool COMMENT 'Whether a page fault was a major fault' CODEC(ZSTD(1)),
    `address` UInt64 COMMENT 'Faulting address for page fault events' CODEC(ZSTD(1)),
    `pages` UInt64 COMMENT 'Number of pages for swap events' CODEC(ZSTD(1)),
    `rw` UInt8 COMMENT 'Read (0) or write (1) for disk I/O' CODEC(ZSTD(1)),
    `queue_depth` UInt32 COMMENT 'Block device queue depth at time of I/O' CODEC(ZSTD(1)),
    `device_id` UInt32 COMMENT 'Block device ID (major:minor encoded)' CODEC(ZSTD(1)),
    `tcp_state` UInt8 COMMENT 'New TCP state after state change' CODEC(ZSTD(1)),
    `tcp_old_state` UInt8 COMMENT 'Previous TCP state before state change' CODEC(ZSTD(1)),
    `tcp_srtt_us` UInt32 COMMENT 'Smoothed RTT in microseconds' CODEC(ZSTD(1)),
    `tcp_cwnd` UInt32 COMMENT 'Congestion window size' CODEC(ZSTD(1)),
    `exit_code` UInt32 COMMENT 'Process exit code' CODEC(ZSTD(1)),
    `target_pid` UInt32 COMMENT 'Target PID for OOM kill events' CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the node running the observoor agent',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name (mainnet, holesky, etc.)'
)
ENGINE = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY (meta_network_name, toYYYYMM(wallclock_slot_start_date_time))
ORDER BY (meta_network_name, wallclock_slot_start_date_time, client_type, event_type, pid)
COMMENT 'Raw eBPF events captured from Ethereum client processes, one row per kernel event.';

CREATE TABLE IF NOT EXISTS sched_off_cpu_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated scheduler off-CPU metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS sched_on_cpu_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated scheduler on-CPU metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS sched_runqueue_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated scheduler run queue metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS swap_in_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated swap-in metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS swap_out_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated swap-out metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS sync_state_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) COMMENT 'Version column for ReplacingMergeTree deduplication' CODEC(DoubleDelta, ZSTD(1)),
    `event_time` DateTime64(3) COMMENT 'Time when the sync state was sampled' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Ethereum slot number at sampling time' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) COMMENT 'Wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `cl_syncing` Bool COMMENT 'Whether the consensus layer is syncing' CODEC(ZSTD(1)),
    `el_optimistic` Bool COMMENT 'Whether the execution layer is in optimistic sync mode' CODEC(ZSTD(1)),
    `el_offline` Bool COMMENT 'Whether the execution layer is unreachable' CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the node running the observoor agent',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name (mainnet, holesky, etc.)'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(event_time))
ORDER BY (meta_network_name, event_time, meta_client_name)
COMMENT 'Sync state snapshots for consensus and execution layers';

CREATE TABLE IF NOT EXISTS syscall_epoll_wait_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated epoll_wait syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_fdatasync_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated fdatasync syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_fsync_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated fsync syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_futex_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated futex syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_mmap_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated mmap syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_pwrite_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated pwrite syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_read_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated read syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_write_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `histogram` Tuple( le_1us UInt32, le_10us UInt32, le_100us UInt32, le_1ms UInt32, le_10ms UInt32, le_100ms UInt32, le_1s UInt32, le_10s UInt32, le_100s UInt32, inf UInt32) CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated write syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_cwnd_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `port_label` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label)
COMMENT 'Aggregated TCP congestion window metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_retransmit_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `port_label` LowCardinality(String),
    `direction` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label, direction)
COMMENT 'Aggregated TCP retransmit metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_rtt_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `port_label` LowCardinality(String),
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `min` Float32 CODEC(ZSTD(1)),
    `max` Float32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, port_label)
COMMENT 'Aggregated TCP round-trip time metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_state_change_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `window_start` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `interval_ms` UInt16 CODEC(ZSTD(1)),
    `wallclock_slot` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `pid` UInt32 CODEC(ZSTD(1)),
    `client_type` LowCardinality(String),
    `sampling_mode` LowCardinality(String) DEFAULT 'none',
    `sampling_rate` Float32 DEFAULT 1.,
    `sum` Float32 CODEC(ZSTD(1)),
    `count` UInt32 CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_network_name` LowCardinality(String)
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(window_start))
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type)
COMMENT 'Aggregated TCP state change events from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS block_merge ON CLUSTER '{cluster}'
AS block_merge_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'block_merge_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated block device I/O merge metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS cpu_utilization ON CLUSTER '{cluster}'
AS cpu_utilization_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'cpu_utilization_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated CPU utilization metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS disk_bytes ON CLUSTER '{cluster}'
AS disk_bytes_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'disk_bytes_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated disk I/O byte metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS disk_latency ON CLUSTER '{cluster}'
AS disk_latency_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'disk_latency_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated disk I/O latency metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS disk_queue_depth ON CLUSTER '{cluster}'
AS disk_queue_depth_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'disk_queue_depth_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated disk queue depth metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS fd_close ON CLUSTER '{cluster}'
AS fd_close_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'fd_close_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated file descriptor close metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS fd_open ON CLUSTER '{cluster}'
AS fd_open_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'fd_open_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated file descriptor open metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS host_specs ON CLUSTER '{cluster}'
AS host_specs_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'host_specs_local', cityHash64(event_time, meta_network_name, host_id, meta_client_name))
COMMENT 'Periodic host hardware specification snapshots including CPU, memory, and disk details';

CREATE TABLE IF NOT EXISTS mem_compaction ON CLUSTER '{cluster}'
AS mem_compaction_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'mem_compaction_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated memory compaction metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS mem_reclaim ON CLUSTER '{cluster}'
AS mem_reclaim_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'mem_reclaim_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated memory reclaim metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS memory_usage ON CLUSTER '{cluster}'
AS memory_usage_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'memory_usage_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Periodic memory usage snapshots of Ethereum client processes from /proc/[pid]/status';

CREATE TABLE IF NOT EXISTS net_io ON CLUSTER '{cluster}'
AS net_io_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'net_io_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated network I/O metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS oom_kill ON CLUSTER '{cluster}'
AS oom_kill_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'oom_kill_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated OOM kill events from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS page_fault_major ON CLUSTER '{cluster}'
AS page_fault_major_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'page_fault_major_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated major page fault metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS page_fault_minor ON CLUSTER '{cluster}'
AS page_fault_minor_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'page_fault_minor_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated minor page fault metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS process_exit ON CLUSTER '{cluster}'
AS process_exit_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'process_exit_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated process exit events from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS process_fd_usage ON CLUSTER '{cluster}'
AS process_fd_usage_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'process_fd_usage_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Periodic file descriptor usage snapshots of Ethereum client processes from /proc/[pid]/fd and /proc/[pid]/limits';

CREATE TABLE IF NOT EXISTS process_io_usage ON CLUSTER '{cluster}'
AS process_io_usage_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'process_io_usage_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Periodic I/O usage snapshots of Ethereum client processes from /proc/[pid]/io';

CREATE TABLE IF NOT EXISTS process_sched_usage ON CLUSTER '{cluster}'
AS process_sched_usage_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'process_sched_usage_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Periodic scheduler usage snapshots of Ethereum client processes from /proc/[pid]/status and /proc/[pid]/sched';

CREATE TABLE IF NOT EXISTS raw_events ON CLUSTER '{cluster}'
AS raw_events_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'raw_events_local', rand())
COMMENT 'Raw eBPF events captured from Ethereum client processes, one row per kernel event.';

CREATE TABLE IF NOT EXISTS sched_off_cpu ON CLUSTER '{cluster}'
AS sched_off_cpu_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'sched_off_cpu_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated scheduler off-CPU metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS sched_on_cpu ON CLUSTER '{cluster}'
AS sched_on_cpu_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'sched_on_cpu_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated scheduler on-CPU metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS sched_runqueue ON CLUSTER '{cluster}'
AS sched_runqueue_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'sched_runqueue_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated scheduler run queue metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS swap_in ON CLUSTER '{cluster}'
AS swap_in_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'swap_in_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated swap-in metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS swap_out ON CLUSTER '{cluster}'
AS swap_out_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'swap_out_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated swap-out metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS sync_state ON CLUSTER '{cluster}'
AS sync_state_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'sync_state_local', cityHash64(event_time, meta_network_name, meta_client_name))
COMMENT 'Sync state snapshots for consensus and execution layers';

CREATE TABLE IF NOT EXISTS syscall_epoll_wait ON CLUSTER '{cluster}'
AS syscall_epoll_wait_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_epoll_wait_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated epoll_wait syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_fdatasync ON CLUSTER '{cluster}'
AS syscall_fdatasync_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_fdatasync_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated fdatasync syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_fsync ON CLUSTER '{cluster}'
AS syscall_fsync_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_fsync_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated fsync syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_futex ON CLUSTER '{cluster}'
AS syscall_futex_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_futex_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated futex syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_mmap ON CLUSTER '{cluster}'
AS syscall_mmap_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_mmap_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated mmap syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_pwrite ON CLUSTER '{cluster}'
AS syscall_pwrite_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_pwrite_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated pwrite syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_read ON CLUSTER '{cluster}'
AS syscall_read_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_read_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated read syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS syscall_write ON CLUSTER '{cluster}'
AS syscall_write_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'syscall_write_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated write syscall metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_cwnd ON CLUSTER '{cluster}'
AS tcp_cwnd_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'tcp_cwnd_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated TCP congestion window metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_retransmit ON CLUSTER '{cluster}'
AS tcp_retransmit_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'tcp_retransmit_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated TCP retransmit metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_rtt ON CLUSTER '{cluster}'
AS tcp_rtt_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'tcp_rtt_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated TCP round-trip time metrics from eBPF tracing of Ethereum client processes';

CREATE TABLE IF NOT EXISTS tcp_state_change ON CLUSTER '{cluster}'
AS tcp_state_change_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'tcp_state_change_local', cityHash64(window_start, meta_network_name, meta_client_name))
COMMENT 'Aggregated TCP state change events from eBPF tracing of Ethereum client processes';
