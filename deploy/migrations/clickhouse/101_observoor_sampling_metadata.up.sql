-- Migration 006: add sampling metadata columns to aggregated metric tables.
-- Adds sampling_mode + sampling_rate for downstream extrapolation.

-- syscall_read
ALTER TABLE observoor.syscall_read_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_read ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_write
ALTER TABLE observoor.syscall_write_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_write ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_futex
ALTER TABLE observoor.syscall_futex_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_futex ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_mmap
ALTER TABLE observoor.syscall_mmap_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_mmap ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_epoll_wait
ALTER TABLE observoor.syscall_epoll_wait_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_epoll_wait ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_fsync
ALTER TABLE observoor.syscall_fsync_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_fsync ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_fdatasync
ALTER TABLE observoor.syscall_fdatasync_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_fdatasync ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- syscall_pwrite
ALTER TABLE observoor.syscall_pwrite_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.syscall_pwrite ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- sched_on_cpu
ALTER TABLE observoor.sched_on_cpu_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.sched_on_cpu ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- sched_off_cpu
ALTER TABLE observoor.sched_off_cpu_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.sched_off_cpu ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- sched_runqueue
ALTER TABLE observoor.sched_runqueue_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.sched_runqueue ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- mem_reclaim
ALTER TABLE observoor.mem_reclaim_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.mem_reclaim ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- mem_compaction
ALTER TABLE observoor.mem_compaction_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.mem_compaction ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- disk_latency
ALTER TABLE observoor.disk_latency_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.disk_latency ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- page_fault_major
ALTER TABLE observoor.page_fault_major_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.page_fault_major ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- page_fault_minor
ALTER TABLE observoor.page_fault_minor_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.page_fault_minor ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- swap_in
ALTER TABLE observoor.swap_in_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.swap_in ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- swap_out
ALTER TABLE observoor.swap_out_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.swap_out ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- oom_kill
ALTER TABLE observoor.oom_kill_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.oom_kill ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- fd_open
ALTER TABLE observoor.fd_open_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.fd_open ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- fd_close
ALTER TABLE observoor.fd_close_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.fd_close ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- process_exit
ALTER TABLE observoor.process_exit_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.process_exit ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- tcp_state_change
ALTER TABLE observoor.tcp_state_change_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.tcp_state_change ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- net_io
ALTER TABLE observoor.net_io_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.net_io ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- tcp_retransmit
ALTER TABLE observoor.tcp_retransmit_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.tcp_retransmit ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- disk_bytes
ALTER TABLE observoor.disk_bytes_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.disk_bytes ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- block_merge
ALTER TABLE observoor.block_merge_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.block_merge ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- tcp_rtt
ALTER TABLE observoor.tcp_rtt_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.tcp_rtt ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- tcp_cwnd
ALTER TABLE observoor.tcp_cwnd_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.tcp_cwnd ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- disk_queue_depth
ALTER TABLE observoor.disk_queue_depth_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.disk_queue_depth ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

-- cpu_utilization
ALTER TABLE observoor.cpu_utilization_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;

ALTER TABLE observoor.cpu_utilization ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS sampling_mode LowCardinality(String) DEFAULT 'none' AFTER client_type,
    ADD COLUMN IF NOT EXISTS sampling_rate Float32 DEFAULT 1.0 AFTER sampling_mode;
