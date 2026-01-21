-- Migration 087 down: Remove peer_id columns from all libp2p tables

-- ============================================
-- SECTION 1: Remove peer_id columns from local tables (27 tables)
-- ============================================

ALTER TABLE libp2p_add_peer_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_remove_peer_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_recv_rpc_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_send_rpc_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_drop_rpc_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_join_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_leave_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_graft_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_prune_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_deliver_message_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_reject_message_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_duplicate_message_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_aggregate_and_proof_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_message_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_idontwant_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_data_column_custody_probe_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;

-- ============================================
-- SECTION 2: Remove peer_id columns from distributed tables (27 tables)
-- ============================================

ALTER TABLE libp2p_add_peer ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_remove_peer ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_recv_rpc ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_send_rpc ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_drop_rpc ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_join ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_leave ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_graft ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_prune ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_deliver_message ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_reject_message ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_duplicate_message ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_handle_status ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_handle_metadata ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_aggregate_and_proof ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_message ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_subscription ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_ihave ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_iwant ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_graft ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_meta_control_idontwant ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;
ALTER TABLE libp2p_rpc_data_column_custody_probe ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS peer_id;

-- ============================================
-- SECTION 3: Remove remote_peer_id columns from local tables (3 tables)
-- ============================================

ALTER TABLE libp2p_connected_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS remote_peer_id;
ALTER TABLE libp2p_disconnected_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS remote_peer_id;
ALTER TABLE libp2p_synthetic_heartbeat_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS remote_peer_id;

-- ============================================
-- SECTION 4: Remove remote_peer_id columns from distributed tables (3 tables)
-- ============================================

ALTER TABLE libp2p_connected ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS remote_peer_id;
ALTER TABLE libp2p_disconnected ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS remote_peer_id;
ALTER TABLE libp2p_synthetic_heartbeat ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS remote_peer_id;

-- ============================================
-- SECTION 5: Remove graft_peer_id column from local table (1 table)
-- ============================================

ALTER TABLE libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS graft_peer_id;

-- ============================================
-- SECTION 6: Remove graft_peer_id column from distributed table (1 table)
-- ============================================

ALTER TABLE libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}' DROP COLUMN IF EXISTS graft_peer_id;
