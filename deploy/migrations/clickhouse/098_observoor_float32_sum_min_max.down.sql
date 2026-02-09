-- Migration 003 rollback: Float32 → Int64 for sum/min/max columns

--------------------------------------------------------------------------------
-- LATENCY TABLES (14) — restore sum, min, max to Int64
--------------------------------------------------------------------------------

-- syscall_read
ALTER TABLE observoor.syscall_read_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_read ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_write
ALTER TABLE observoor.syscall_write_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_write ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_futex
ALTER TABLE observoor.syscall_futex_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_futex ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_mmap
ALTER TABLE observoor.syscall_mmap_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_mmap ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_epoll_wait
ALTER TABLE observoor.syscall_epoll_wait_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_epoll_wait ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_fsync
ALTER TABLE observoor.syscall_fsync_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_fsync ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_fdatasync
ALTER TABLE observoor.syscall_fdatasync_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_fdatasync ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- syscall_pwrite
ALTER TABLE observoor.syscall_pwrite_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.syscall_pwrite ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- sched_on_cpu
ALTER TABLE observoor.sched_on_cpu_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.sched_on_cpu ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- sched_off_cpu
ALTER TABLE observoor.sched_off_cpu_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.sched_off_cpu ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- sched_runqueue
ALTER TABLE observoor.sched_runqueue_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.sched_runqueue ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- mem_reclaim
ALTER TABLE observoor.mem_reclaim_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.mem_reclaim ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- mem_compaction
ALTER TABLE observoor.mem_compaction_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.mem_compaction ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- disk_latency
ALTER TABLE observoor.disk_latency_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.disk_latency ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;


--------------------------------------------------------------------------------
-- COUNTER TABLES (13) — restore sum to Int64
--------------------------------------------------------------------------------

-- page_fault_major
ALTER TABLE observoor.page_fault_major_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.page_fault_major ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- page_fault_minor
ALTER TABLE observoor.page_fault_minor_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.page_fault_minor ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- swap_in
ALTER TABLE observoor.swap_in_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.swap_in ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- swap_out
ALTER TABLE observoor.swap_out_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.swap_out ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- oom_kill
ALTER TABLE observoor.oom_kill_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.oom_kill ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- fd_open
ALTER TABLE observoor.fd_open_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.fd_open ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- fd_close
ALTER TABLE observoor.fd_close_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.fd_close ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- process_exit
ALTER TABLE observoor.process_exit_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.process_exit ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- tcp_state_change
ALTER TABLE observoor.tcp_state_change_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_state_change ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- net_io
ALTER TABLE observoor.net_io_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.net_io ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- tcp_retransmit
ALTER TABLE observoor.tcp_retransmit_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_retransmit ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- disk_bytes
ALTER TABLE observoor.disk_bytes_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.disk_bytes ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;

-- block_merge
ALTER TABLE observoor.block_merge_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.block_merge ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64;


--------------------------------------------------------------------------------
-- GAUGE TABLES (3) — restore sum, min, max to Int64
--------------------------------------------------------------------------------

-- tcp_rtt
ALTER TABLE observoor.tcp_rtt_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_rtt ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- tcp_cwnd
ALTER TABLE observoor.tcp_cwnd_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.tcp_cwnd ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;

-- disk_queue_depth
ALTER TABLE observoor.disk_queue_depth_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `min` Int64 CODEC(ZSTD(1)),
    MODIFY COLUMN `max` Int64 CODEC(ZSTD(1));

ALTER TABLE observoor.disk_queue_depth ON CLUSTER '{cluster}'
    MODIFY COLUMN `sum` Int64,
    MODIFY COLUMN `min` Int64,
    MODIFY COLUMN `max` Int64;
