-- Migration 003: Int64 → Float32 for sum/min/max columns
-- Reduces compressed storage ~48% with negligible precision loss for monitoring data.
-- ALTER MODIFY COLUMN runs as a background mutation. No downtime required.

--------------------------------------------------------------------------------
-- LATENCY TABLES (14) — modify sum, min, max
--------------------------------------------------------------------------------

-- syscall_read
ALTER TABLE observoor.syscall_read_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_read ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_write
ALTER TABLE observoor.syscall_write_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_write ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_futex
ALTER TABLE observoor.syscall_futex_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_futex ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_mmap
ALTER TABLE observoor.syscall_mmap_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_mmap ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_epoll_wait
ALTER TABLE observoor.syscall_epoll_wait_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_epoll_wait ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_fsync
ALTER TABLE observoor.syscall_fsync_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_fsync ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_fdatasync
ALTER TABLE observoor.syscall_fdatasync_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_fdatasync ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- syscall_pwrite
ALTER TABLE observoor.syscall_pwrite_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_pwrite ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- sched_on_cpu
ALTER TABLE observoor.sched_on_cpu_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.sched_on_cpu ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- sched_off_cpu
ALTER TABLE observoor.sched_off_cpu_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.sched_off_cpu ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- sched_runqueue
ALTER TABLE observoor.sched_runqueue_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.sched_runqueue ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- mem_reclaim
ALTER TABLE observoor.mem_reclaim_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.mem_reclaim ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- mem_compaction
ALTER TABLE observoor.mem_compaction_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.mem_compaction ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- disk_latency
ALTER TABLE observoor.disk_latency_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.disk_latency ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;


--------------------------------------------------------------------------------
-- COUNTER TABLES (13) — modify sum only
--------------------------------------------------------------------------------

-- page_fault_major
ALTER TABLE observoor.page_fault_major_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.page_fault_major ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- page_fault_minor
ALTER TABLE observoor.page_fault_minor_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.page_fault_minor ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- swap_in
ALTER TABLE observoor.swap_in_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.swap_in ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- swap_out
ALTER TABLE observoor.swap_out_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.swap_out ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- oom_kill
ALTER TABLE observoor.oom_kill_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.oom_kill ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- fd_open
ALTER TABLE observoor.fd_open_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.fd_open ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- fd_close
ALTER TABLE observoor.fd_close_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.fd_close ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- process_exit
ALTER TABLE observoor.process_exit_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.process_exit ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- tcp_state_change
ALTER TABLE observoor.tcp_state_change_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_state_change ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- net_io
ALTER TABLE observoor.net_io_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.net_io ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- tcp_retransmit
ALTER TABLE observoor.tcp_retransmit_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_retransmit ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- disk_bytes
ALTER TABLE observoor.disk_bytes_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.disk_bytes ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;

-- block_merge
ALTER TABLE observoor.block_merge_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.block_merge ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32;


--------------------------------------------------------------------------------
-- GAUGE TABLES (3) — modify sum, min, max
--------------------------------------------------------------------------------

-- tcp_rtt
ALTER TABLE observoor.tcp_rtt_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_rtt ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- tcp_cwnd
ALTER TABLE observoor.tcp_cwnd_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_cwnd ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;

-- disk_queue_depth
ALTER TABLE observoor.disk_queue_depth_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Float32 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Float32 CODEC(ZSTD(1));

ALTER TABLE observoor.disk_queue_depth ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Float32,
    MODIFY COLUMN `min` Float32,
    MODIFY COLUMN `max` Float32;
