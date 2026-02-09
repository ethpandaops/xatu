DROP DATABASE IF EXISTS observoor ON CLUSTER '{cluster}';

-- Observoor ClickHouse Schema Teardown

-- Raw events
DROP TABLE IF EXISTS observoor.raw_events ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.raw_events_local ON CLUSTER '{cluster}';

-- Sync state
DROP TABLE IF EXISTS observoor.sync_state ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.sync_state_local ON CLUSTER '{cluster}';

-- Syscall latency tables
DROP TABLE IF EXISTS observoor.syscall_read ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_read_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_write ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_write_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_futex ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_futex_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_mmap ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_mmap_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_epoll_wait ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_epoll_wait_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_fsync ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_fsync_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_fdatasync ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_fdatasync_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_pwrite ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.syscall_pwrite_local ON CLUSTER '{cluster}';

-- Scheduler latency tables
DROP TABLE IF EXISTS observoor.sched_on_cpu ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.sched_on_cpu_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.sched_off_cpu ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.sched_off_cpu_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.sched_runqueue ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.sched_runqueue_local ON CLUSTER '{cluster}';

-- Memory latency tables
DROP TABLE IF EXISTS observoor.mem_reclaim ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.mem_reclaim_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.mem_compaction ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.mem_compaction_local ON CLUSTER '{cluster}';

-- Disk latency table
DROP TABLE IF EXISTS observoor.disk_latency ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.disk_latency_local ON CLUSTER '{cluster}';

-- Memory counter tables
DROP TABLE IF EXISTS observoor.page_fault_major ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.page_fault_major_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.page_fault_minor ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.page_fault_minor_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.swap_in ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.swap_in_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.swap_out ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.swap_out_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.oom_kill ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.oom_kill_local ON CLUSTER '{cluster}';

-- Process counter tables
DROP TABLE IF EXISTS observoor.fd_open ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.fd_open_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.fd_close ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.fd_close_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.process_exit ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.process_exit_local ON CLUSTER '{cluster}';

-- Network counter tables
DROP TABLE IF EXISTS observoor.tcp_state_change ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.tcp_state_change_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.net_io ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.net_io_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.tcp_retransmit ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.tcp_retransmit_local ON CLUSTER '{cluster}';

-- Disk counter tables
DROP TABLE IF EXISTS observoor.disk_bytes ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.disk_bytes_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.block_merge ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.block_merge_local ON CLUSTER '{cluster}';

-- Gauge tables
DROP TABLE IF EXISTS observoor.tcp_rtt ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.tcp_rtt_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.tcp_cwnd ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.tcp_cwnd_local ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.disk_queue_depth ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS observoor.disk_queue_depth_local ON CLUSTER '{cluster}';
