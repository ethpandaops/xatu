-- Migration 102: Schema V2 (GENERATED) - DOWN
--
-- Drops all objects created by 001_init.up.sql in reverse dependency order:
--   1. Materialized Views
--   2. Distributed Tables
--   3. Local Tables
--   4. Databases

-- -----------------------------------------------------------------------------
-- 1. Materialized Views
-- -----------------------------------------------------------------------------

DROP VIEW IF EXISTS default.beacon_api_slot_attestation_mv_local ON CLUSTER '{cluster}' SYNC;
DROP VIEW IF EXISTS default.beacon_api_slot_block_mv_local ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 2. Distributed Tables (default)
-- -----------------------------------------------------------------------------

DROP TABLE IF EXISTS default.beacon_api_eth_v1_beacon_blob ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_attestation ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_block_gossip ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v3_validator_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_slot ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_block_classification ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.blob_submitter ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.block_native_mempool_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_bls_to_execution_change ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_deposit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_proposer_slashing ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_sync_aggregate ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_committee ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_proposer_duty ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_sync_committee ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators_pubkeys ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators_withdrawal_credentials ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_address_appearances ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_balance_diffs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_balance_reads ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_contracts ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_erc20_transfers ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_erc721_transfers ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_four_byte_counts ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_logs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_native_transfers ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_nonce_diffs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_nonce_reads ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_storage_diffs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_storage_reads ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_traces ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog_agg ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.consensus_engine_api_get_blobs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.consensus_engine_api_new_payload ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.ethseer_validator_entity ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_block_metrics ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_engine_get_blobs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_engine_new_payload ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_state_size ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.imported_sources ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_add_peer ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_connected ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_deliver_message ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_disconnected ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_drop_rpc ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_duplicate_message ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_aggregate_and_proof ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_graft ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_handle_metadata ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_handle_status ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_identify ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_join ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_leave ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_peer ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_prune ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_publish_message ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_recv_rpc ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_reject_message ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_remove_peer ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_data_column_custody_probe ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_graft ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_idontwant ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_ihave ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_iwant ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_message ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_subscription ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_send_rpc ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_synthetic_heartbeat ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mempool_dumpster_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mempool_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mev_relay_bid_trace ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mev_relay_proposer_payload_delivered ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mev_relay_validator_registration ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.node_record_consensus ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.node_record_execution ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 2. Distributed Tables (observoor)
-- -----------------------------------------------------------------------------

DROP TABLE IF EXISTS observoor.block_merge ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.cpu_utilization ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.disk_bytes ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.disk_latency ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.disk_queue_depth ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.fd_close ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.fd_open ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.host_specs ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.mem_compaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.mem_reclaim ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.memory_usage ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.net_io ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.oom_kill ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.page_fault_major ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.page_fault_minor ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_exit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_fd_usage ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_io_usage ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_sched_usage ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.raw_events ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sched_off_cpu ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sched_on_cpu ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sched_runqueue ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.swap_in ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.swap_out ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sync_state ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_epoll_wait ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_fdatasync ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_fsync ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_futex ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_mmap ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_pwrite ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_read ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_write ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_cwnd ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_retransmit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_rtt ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_state_change ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 2. Distributed Tables (admin)
-- -----------------------------------------------------------------------------

DROP TABLE IF EXISTS admin.cryo ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS admin.execution_block ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 3. Local Tables (default)
-- -----------------------------------------------------------------------------

DROP TABLE IF EXISTS default.beacon_api_eth_v1_beacon_blob_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_attestation_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_block_gossip_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_proposer_duty_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_eth_v3_validator_block_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_api_slot_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.beacon_block_classification_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.blob_submitter_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.block_native_mempool_transaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_attester_slashing_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_bls_to_execution_change_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_deposit_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_proposer_slashing_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_sync_aggregate_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_voluntary_exit_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_committee_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_proposer_duty_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_sync_committee_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators_pubkeys_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators_withdrawal_credentials_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_address_appearances_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_balance_diffs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_balance_reads_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_block_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_contracts_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_erc20_transfers_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_erc721_transfers_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_four_byte_counts_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_logs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_native_transfers_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_nonce_diffs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_nonce_reads_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_storage_diffs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_storage_reads_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_traces_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog_agg_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.consensus_engine_api_get_blobs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.consensus_engine_api_new_payload_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.ethseer_validator_entity_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_block_metrics_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_engine_get_blobs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_engine_new_payload_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_state_size_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.execution_transaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.imported_sources_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_add_peer_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_connected_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_deliver_message_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_disconnected_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_drop_rpc_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_duplicate_message_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_aggregate_and_proof_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_graft_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_handle_metadata_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_handle_status_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_identify_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_join_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_leave_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_peer_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_prune_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_publish_message_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_recv_rpc_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_reject_message_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_remove_peer_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_data_column_custody_probe_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_idontwant_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_message_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_send_rpc_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.libp2p_synthetic_heartbeat_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mempool_dumpster_transaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mempool_transaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mev_relay_bid_trace_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mev_relay_proposer_payload_delivered_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.mev_relay_validator_registration_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.node_record_consensus_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.node_record_execution_local ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 3. Local Tables (observoor)
-- -----------------------------------------------------------------------------

DROP TABLE IF EXISTS observoor.block_merge_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.cpu_utilization_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.disk_bytes_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.disk_latency_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.disk_queue_depth_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.fd_close_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.fd_open_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.host_specs_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.mem_compaction_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.mem_reclaim_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.memory_usage_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.net_io_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.oom_kill_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.page_fault_major_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.page_fault_minor_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_exit_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_fd_usage_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_io_usage_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.process_sched_usage_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.raw_events_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sched_off_cpu_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sched_on_cpu_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sched_runqueue_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.swap_in_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.swap_out_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.sync_state_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_epoll_wait_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_fdatasync_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_fsync_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_futex_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_mmap_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_pwrite_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_read_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.syscall_write_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_cwnd_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_retransmit_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_rtt_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS observoor.tcp_state_change_local ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 3. Local Tables (admin)
-- -----------------------------------------------------------------------------

DROP TABLE IF EXISTS admin.cryo_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS admin.execution_block_local ON CLUSTER '{cluster}' SYNC;

-- -----------------------------------------------------------------------------
-- 4. Databases
-- -----------------------------------------------------------------------------

DROP DATABASE IF EXISTS observoor ON CLUSTER '{cluster}' SYNC;
DROP DATABASE IF EXISTS admin ON CLUSTER '{cluster}' SYNC;
