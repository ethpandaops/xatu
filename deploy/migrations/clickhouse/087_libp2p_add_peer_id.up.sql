-- Migration 087: Add peer_id columns to all libp2p tables
-- Adds raw peer_id string alongside existing peer_id_unique_key (seahash)
-- Historical rows will have NULL; only new data will be populated

-- ============================================
-- SECTION 1: Add peer_id columns to local tables (27 tables)
-- ============================================

ALTER TABLE libp2p_add_peer_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_remove_peer_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_recv_rpc_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID of the sender' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_send_rpc_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID of the receiver' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_drop_rpc_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_join_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID that joined the topic' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_leave_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID that left the topic' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_graft_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_prune_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_deliver_message_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID that delivered the message' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_reject_message_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_duplicate_message_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_aggregate_and_proof_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_message_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_idontwant_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_data_column_custody_probe_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

-- ============================================
-- SECTION 2: Add peer_id columns to distributed tables (27 tables)
-- ============================================

ALTER TABLE libp2p_add_peer ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_remove_peer ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_recv_rpc ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID of the sender' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_send_rpc ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID of the receiver' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_drop_rpc ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_join ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID that joined the topic' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_leave ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID that left the topic' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_graft ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_prune ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_deliver_message ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID that delivered the message' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_reject_message ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_duplicate_message ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_handle_status ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_handle_metadata ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_aggregate_and_proof ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_message ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_subscription ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_ihave ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_iwant ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_graft ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_meta_control_idontwant ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

ALTER TABLE libp2p_rpc_data_column_custody_probe ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS peer_id Nullable(String) COMMENT 'The libp2p peer ID' CODEC(ZSTD(1)) AFTER peer_id_unique_key;

-- ============================================
-- SECTION 3: Add remote_peer_id columns to local tables (3 tables)
-- ============================================

ALTER TABLE libp2p_connected_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS remote_peer_id Nullable(String) COMMENT 'The remote libp2p peer ID' CODEC(ZSTD(1)) AFTER remote_peer_id_unique_key;

ALTER TABLE libp2p_disconnected_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS remote_peer_id Nullable(String) COMMENT 'The remote libp2p peer ID' CODEC(ZSTD(1)) AFTER remote_peer_id_unique_key;

ALTER TABLE libp2p_synthetic_heartbeat_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS remote_peer_id Nullable(String) COMMENT 'The remote libp2p peer ID' CODEC(ZSTD(1)) AFTER remote_peer_id_unique_key;

-- ============================================
-- SECTION 4: Add remote_peer_id columns to distributed tables (3 tables)
-- ============================================

ALTER TABLE libp2p_connected ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS remote_peer_id Nullable(String) COMMENT 'The remote libp2p peer ID' CODEC(ZSTD(1)) AFTER remote_peer_id_unique_key;

ALTER TABLE libp2p_disconnected ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS remote_peer_id Nullable(String) COMMENT 'The remote libp2p peer ID' CODEC(ZSTD(1)) AFTER remote_peer_id_unique_key;

ALTER TABLE libp2p_synthetic_heartbeat ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS remote_peer_id Nullable(String) COMMENT 'The remote libp2p peer ID' CODEC(ZSTD(1)) AFTER remote_peer_id_unique_key;

-- ============================================
-- SECTION 5: Add graft_peer_id column to local table (1 table)
-- ============================================

ALTER TABLE libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS graft_peer_id Nullable(String) COMMENT 'The graft libp2p peer ID' CODEC(ZSTD(1)) AFTER graft_peer_id_unique_key;

-- ============================================
-- SECTION 6: Add graft_peer_id column to distributed table (1 table)
-- ============================================

ALTER TABLE libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS graft_peer_id Nullable(String) COMMENT 'The graft libp2p peer ID' CODEC(ZSTD(1)) AFTER graft_peer_id_unique_key;
