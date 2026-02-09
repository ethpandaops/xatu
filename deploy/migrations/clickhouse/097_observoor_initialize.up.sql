CREATE DATABASE IF NOT EXISTS observoor ON CLUSTER '{cluster}';

--------------------------------------------------------------------------------
-- RAW EVENTS
-- One row per kernel event. Use for debugging and detailed analysis.
-- Expected volume: ~20 GB/day at 36k events/sec
--------------------------------------------------------------------------------

CREATE TABLE observoor.raw_events_local ON CLUSTER '{cluster}' (
    -- Timing
    timestamp_ns UInt64 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot UInt64 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),

    -- Sync state
    cl_syncing Bool CODEC(ZSTD(1)),
    el_optimistic Bool CODEC(ZSTD(1)),
    el_offline Bool CODEC(ZSTD(1)),

    -- Process identification
    pid UInt32 CODEC(ZSTD(1)),
    tid UInt32 CODEC(ZSTD(1)),
    event_type LowCardinality(String),
    client_type LowCardinality(String),

    -- Common fields
    latency_ns UInt64 CODEC(ZSTD(1)),
    bytes Int64 CODEC(ZSTD(1)),

    -- Network fields
    src_port UInt16 CODEC(ZSTD(1)),
    dst_port UInt16 CODEC(ZSTD(1)),

    -- File descriptor fields
    fd Int32 CODEC(ZSTD(1)),
    filename String CODEC(ZSTD(1)),

    -- Scheduler fields
    voluntary Bool CODEC(ZSTD(1)),
    on_cpu_ns UInt64 CODEC(ZSTD(1)),
    runqueue_ns UInt64 CODEC(ZSTD(1)),
    off_cpu_ns UInt64 CODEC(ZSTD(1)),

    -- Memory fields
    major Bool CODEC(ZSTD(1)),
    address UInt64 CODEC(ZSTD(1)),
    pages UInt64 CODEC(ZSTD(1)),

    -- Disk I/O fields
    rw UInt8 CODEC(ZSTD(1)),
    queue_depth UInt32 CODEC(ZSTD(1)),
    device_id UInt32 CODEC(ZSTD(1)),

    -- TCP fields
    tcp_state UInt8 CODEC(ZSTD(1)),
    tcp_old_state UInt8 CODEC(ZSTD(1)),
    tcp_srtt_us UInt32 CODEC(ZSTD(1)),
    tcp_cwnd UInt32 CODEC(ZSTD(1)),

    -- Process lifecycle fields
    exit_code UInt32 CODEC(ZSTD(1)),
    target_pid UInt32 CODEC(ZSTD(1)),

    -- Metadata
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}'
)
PARTITION BY toStartOfMonth(wallclock_slot_start_date_time)
ORDER BY (meta_network_name, wallclock_slot_start_date_time, client_type, event_type, pid);

ALTER TABLE observoor.raw_events_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Raw eBPF events captured from Ethereum client processes, one row per kernel event.',
COMMENT COLUMN timestamp_ns 'Wall clock time of the event in nanoseconds since Unix epoch',
COMMENT COLUMN wallclock_slot 'Ethereum slot number at the time of the event (from wall clock)',
COMMENT COLUMN wallclock_slot_start_date_time 'Wall clock time when the slot started',
COMMENT COLUMN cl_syncing 'Whether the consensus layer was syncing when this event was captured',
COMMENT COLUMN el_optimistic 'Whether the execution layer was in optimistic sync mode when this event was captured',
COMMENT COLUMN el_offline 'Whether the execution layer was unreachable when this event was captured',
COMMENT COLUMN pid 'Process ID of the traced Ethereum client',
COMMENT COLUMN tid 'Thread ID within the traced process',
COMMENT COLUMN event_type 'Type of eBPF event (syscall_read, disk_io, net_tx, etc.)',
COMMENT COLUMN client_type 'Ethereum client implementation (geth, reth, prysm, lighthouse, etc.)',
COMMENT COLUMN latency_ns 'Latency in nanoseconds for syscall and disk I/O events',
COMMENT COLUMN bytes 'Byte count for I/O events',
COMMENT COLUMN src_port 'Source port for network events',
COMMENT COLUMN dst_port 'Destination port for network events',
COMMENT COLUMN fd 'File descriptor number',
COMMENT COLUMN filename 'Filename for fd_open events',
COMMENT COLUMN voluntary 'Whether a context switch was voluntary',
COMMENT COLUMN on_cpu_ns 'Time spent on CPU in nanoseconds before a context switch',
COMMENT COLUMN runqueue_ns 'Time spent waiting in the run queue',
COMMENT COLUMN off_cpu_ns 'Time spent off CPU',
COMMENT COLUMN major 'Whether a page fault was a major fault',
COMMENT COLUMN address 'Faulting address for page fault events',
COMMENT COLUMN pages 'Number of pages for swap events',
COMMENT COLUMN rw 'Read (0) or write (1) for disk I/O',
COMMENT COLUMN queue_depth 'Block device queue depth at time of I/O',
COMMENT COLUMN device_id 'Block device ID (major:minor encoded)',
COMMENT COLUMN tcp_state 'New TCP state after state change',
COMMENT COLUMN tcp_old_state 'Previous TCP state before state change',
COMMENT COLUMN tcp_srtt_us 'Smoothed RTT in microseconds',
COMMENT COLUMN tcp_cwnd 'Congestion window size',
COMMENT COLUMN exit_code 'Process exit code',
COMMENT COLUMN target_pid 'Target PID for OOM kill events',
COMMENT COLUMN meta_client_name 'Name of the node running the observoor agent',
COMMENT COLUMN meta_network_name 'Ethereum network name (mainnet, holesky, etc.)';

CREATE TABLE observoor.raw_events ON CLUSTER '{cluster}' AS observoor.raw_events_local
ENGINE = Distributed('{cluster}', 'observoor', raw_events_local, rand());


--------------------------------------------------------------------------------
-- SYNC STATE
-- Separate table for consensus/execution layer sync state.
-- Polled periodically (e.g., every slot) to reduce per-row overhead.
--------------------------------------------------------------------------------

CREATE TABLE observoor.sync_state_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    event_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    cl_syncing Bool CODEC(ZSTD(1)),
    el_optimistic Bool CODEC(ZSTD(1)),
    el_offline Bool CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(event_time)
ORDER BY (meta_network_name, event_time, meta_client_name);

ALTER TABLE observoor.sync_state_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Sync state snapshots for consensus and execution layers.',
COMMENT COLUMN updated_date_time 'Version column for ReplacingMergeTree deduplication',
COMMENT COLUMN event_time 'Time when the sync state was sampled',
COMMENT COLUMN wallclock_slot 'Ethereum slot number at sampling time',
COMMENT COLUMN wallclock_slot_start_date_time 'Wall clock time when the slot started',
COMMENT COLUMN cl_syncing 'Whether the consensus layer is syncing',
COMMENT COLUMN el_optimistic 'Whether the execution layer is in optimistic sync mode',
COMMENT COLUMN el_offline 'Whether the execution layer is unreachable',
COMMENT COLUMN meta_client_name 'Name of the node running the observoor agent',
COMMENT COLUMN meta_network_name 'Ethereum network name (mainnet, holesky, etc.)';

CREATE TABLE observoor.sync_state ON CLUSTER '{cluster}' AS observoor.sync_state_local
ENGINE = Distributed('{cluster}', 'observoor', sync_state_local, cityHash64(event_time, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- AGGREGATED METRICS: ONE TABLE PER METRIC
-- Time-windowed aggregations. No metric_name column - the table IS the metric.
-- ReplicatedReplacingMergeTree for idempotent writes, ORDER BY as composite key.
--------------------------------------------------------------------------------


--------------------------------------------------------------------------------
-- SYSCALL LATENCY TABLES (8 tables)
-- Latency histograms for system call operations
--------------------------------------------------------------------------------

CREATE TABLE observoor.syscall_read_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_read ON CLUSTER '{cluster}' AS observoor.syscall_read_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_read_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_write_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_write ON CLUSTER '{cluster}' AS observoor.syscall_write_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_write_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_futex_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_futex ON CLUSTER '{cluster}' AS observoor.syscall_futex_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_futex_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_mmap_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_mmap ON CLUSTER '{cluster}' AS observoor.syscall_mmap_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_mmap_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_epoll_wait_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_epoll_wait ON CLUSTER '{cluster}' AS observoor.syscall_epoll_wait_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_epoll_wait_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_fsync_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_fsync ON CLUSTER '{cluster}' AS observoor.syscall_fsync_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_fsync_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_fdatasync_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_fdatasync ON CLUSTER '{cluster}' AS observoor.syscall_fdatasync_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_fdatasync_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.syscall_pwrite_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.syscall_pwrite ON CLUSTER '{cluster}' AS observoor.syscall_pwrite_local
ENGINE = Distributed('{cluster}', 'observoor', syscall_pwrite_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- SCHEDULER LATENCY TABLES (3 tables)
-- CPU scheduling latency histograms
--------------------------------------------------------------------------------

CREATE TABLE observoor.sched_on_cpu_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.sched_on_cpu ON CLUSTER '{cluster}' AS observoor.sched_on_cpu_local
ENGINE = Distributed('{cluster}', 'observoor', sched_on_cpu_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.sched_off_cpu_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.sched_off_cpu ON CLUSTER '{cluster}' AS observoor.sched_off_cpu_local
ENGINE = Distributed('{cluster}', 'observoor', sched_off_cpu_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.sched_runqueue_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.sched_runqueue ON CLUSTER '{cluster}' AS observoor.sched_runqueue_local
ENGINE = Distributed('{cluster}', 'observoor', sched_runqueue_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- MEMORY LATENCY TABLES (2 tables)
-- Memory reclaim and compaction latency histograms
--------------------------------------------------------------------------------

CREATE TABLE observoor.mem_reclaim_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.mem_reclaim ON CLUSTER '{cluster}' AS observoor.mem_reclaim_local
ENGINE = Distributed('{cluster}', 'observoor', mem_reclaim_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.mem_compaction_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.mem_compaction ON CLUSTER '{cluster}' AS observoor.mem_compaction_local
ENGINE = Distributed('{cluster}', 'observoor', mem_compaction_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- DISK LATENCY TABLE (1 table)
-- Block I/O latency histogram with device and rw dimensions
--------------------------------------------------------------------------------

CREATE TABLE observoor.disk_latency_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    device_id UInt32 CODEC(ZSTD(1)),
    rw LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    histogram Tuple(
        le_1us UInt32,
        le_10us UInt32,
        le_100us UInt32,
        le_1ms UInt32,
        le_10ms UInt32,
        le_100ms UInt32,
        le_1s UInt32,
        le_10s UInt32,
        le_100s UInt32,
        inf UInt32
    ) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw);

CREATE TABLE observoor.disk_latency ON CLUSTER '{cluster}' AS observoor.disk_latency_local
ENGINE = Distributed('{cluster}', 'observoor', disk_latency_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- COUNTER TABLES - MEMORY (5 tables)
-- Simple count/sum aggregations for memory events
--------------------------------------------------------------------------------

CREATE TABLE observoor.page_fault_major_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.page_fault_major ON CLUSTER '{cluster}' AS observoor.page_fault_major_local
ENGINE = Distributed('{cluster}', 'observoor', page_fault_major_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.page_fault_minor_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.page_fault_minor ON CLUSTER '{cluster}' AS observoor.page_fault_minor_local
ENGINE = Distributed('{cluster}', 'observoor', page_fault_minor_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.swap_in_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.swap_in ON CLUSTER '{cluster}' AS observoor.swap_in_local
ENGINE = Distributed('{cluster}', 'observoor', swap_in_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.swap_out_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.swap_out ON CLUSTER '{cluster}' AS observoor.swap_out_local
ENGINE = Distributed('{cluster}', 'observoor', swap_out_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.oom_kill_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.oom_kill ON CLUSTER '{cluster}' AS observoor.oom_kill_local
ENGINE = Distributed('{cluster}', 'observoor', oom_kill_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- COUNTER TABLES - PROCESS (3 tables)
-- File descriptor and process exit counters
--------------------------------------------------------------------------------

CREATE TABLE observoor.fd_open_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.fd_open ON CLUSTER '{cluster}' AS observoor.fd_open_local
ENGINE = Distributed('{cluster}', 'observoor', fd_open_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.fd_close_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.fd_close ON CLUSTER '{cluster}' AS observoor.fd_close_local
ENGINE = Distributed('{cluster}', 'observoor', fd_close_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.process_exit_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.process_exit ON CLUSTER '{cluster}' AS observoor.process_exit_local
ENGINE = Distributed('{cluster}', 'observoor', process_exit_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- COUNTER TABLES - NETWORK (3 tables)
-- TCP state changes, network I/O, and retransmits
--------------------------------------------------------------------------------

CREATE TABLE observoor.tcp_state_change_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type);

CREATE TABLE observoor.tcp_state_change ON CLUSTER '{cluster}' AS observoor.tcp_state_change_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_state_change_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.net_io_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    direction LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
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

CREATE TABLE observoor.net_io ON CLUSTER '{cluster}' AS observoor.net_io_local
ENGINE = Distributed('{cluster}', 'observoor', net_io_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.tcp_retransmit_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    direction LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
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

CREATE TABLE observoor.tcp_retransmit ON CLUSTER '{cluster}' AS observoor.tcp_retransmit_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_retransmit_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- COUNTER TABLES - DISK (2 tables)
-- Disk bytes throughput and block merge counters
--------------------------------------------------------------------------------

CREATE TABLE observoor.disk_bytes_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    device_id UInt32 CODEC(ZSTD(1)),
    rw LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw);

CREATE TABLE observoor.disk_bytes ON CLUSTER '{cluster}' AS observoor.disk_bytes_local
ENGINE = Distributed('{cluster}', 'observoor', disk_bytes_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.block_merge_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    device_id UInt32 CODEC(ZSTD(1)),
    rw LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw);

CREATE TABLE observoor.block_merge ON CLUSTER '{cluster}' AS observoor.block_merge_local
ENGINE = Distributed('{cluster}', 'observoor', block_merge_local, cityHash64(window_start, meta_network_name, meta_client_name));


--------------------------------------------------------------------------------
-- GAUGE TABLES (3 tables)
-- Sampled values with min/max/sum/count (no histogram)
--------------------------------------------------------------------------------

CREATE TABLE observoor.tcp_rtt_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, local_port);

CREATE TABLE observoor.tcp_rtt ON CLUSTER '{cluster}' AS observoor.tcp_rtt_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_rtt_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.tcp_cwnd_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    local_port UInt16 CODEC(ZSTD(1)),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, local_port);

CREATE TABLE observoor.tcp_cwnd ON CLUSTER '{cluster}' AS observoor.tcp_cwnd_local
ENGINE = Distributed('{cluster}', 'observoor', tcp_cwnd_local, cityHash64(window_start, meta_network_name, meta_client_name));

CREATE TABLE observoor.disk_queue_depth_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    window_start DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    interval_ms UInt16 CODEC(ZSTD(1)),
    wallclock_slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    pid UInt32 CODEC(ZSTD(1)),
    client_type LowCardinality(String),
    device_id UInt32 CODEC(ZSTD(1)),
    rw LowCardinality(String),
    sum Int64 CODEC(ZSTD(1)),
    count UInt32 CODEC(ZSTD(1)),
    min Int64 CODEC(ZSTD(1)),
    max Int64 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_name LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(window_start)
ORDER BY (meta_network_name, window_start, meta_client_name, pid, client_type, device_id, rw);

CREATE TABLE observoor.disk_queue_depth ON CLUSTER '{cluster}' AS observoor.disk_queue_depth_local
ENGINE = Distributed('{cluster}', 'observoor', disk_queue_depth_local, cityHash64(window_start, meta_network_name, meta_client_name));
