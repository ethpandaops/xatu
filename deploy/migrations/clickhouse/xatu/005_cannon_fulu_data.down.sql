-- Drop cannon Electra/Fulu beacon data tables.

-- pending_consolidation
DROP TABLE IF EXISTS canonical_beacon_state_pending_consolidation ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_pending_consolidation_local ON CLUSTER '{cluster}' SYNC;

-- pending_partial_withdrawal
DROP TABLE IF EXISTS canonical_beacon_state_pending_partial_withdrawal ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_pending_partial_withdrawal_local ON CLUSTER '{cluster}' SYNC;

-- pending_deposit
DROP TABLE IF EXISTS canonical_beacon_state_pending_deposit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_pending_deposit_local ON CLUSTER '{cluster}' SYNC;

-- finality_checkpoint
DROP TABLE IF EXISTS canonical_beacon_state_finality_checkpoint ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_finality_checkpoint_local ON CLUSTER '{cluster}' SYNC;

-- randao
DROP TABLE IF EXISTS canonical_beacon_state_randao ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_randao_local ON CLUSTER '{cluster}' SYNC;

-- sync_committee_reward
DROP TABLE IF EXISTS canonical_beacon_sync_committee_reward ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_sync_committee_reward_local ON CLUSTER '{cluster}' SYNC;

-- attestation_reward
DROP TABLE IF EXISTS canonical_beacon_attestation_reward ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_attestation_reward_local ON CLUSTER '{cluster}' SYNC;

-- block_reward
DROP TABLE IF EXISTS canonical_beacon_block_reward ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_reward_local ON CLUSTER '{cluster}' SYNC;

-- execution_request_consolidation
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_consolidation ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_consolidation_local ON CLUSTER '{cluster}' SYNC;

-- execution_request_withdrawal
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_withdrawal ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_withdrawal_local ON CLUSTER '{cluster}' SYNC;

-- execution_request_deposit
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_deposit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_deposit_local ON CLUSTER '{cluster}' SYNC;
