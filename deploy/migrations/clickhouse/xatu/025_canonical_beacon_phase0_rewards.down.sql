-- phase0_block_reward
DROP TABLE IF EXISTS canonical_beacon_phase0_block_reward ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_phase0_block_reward_local ON CLUSTER '{cluster}' SYNC;

-- phase0_attestation_reward
DROP TABLE IF EXISTS canonical_beacon_phase0_attestation_reward ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_phase0_attestation_reward_local ON CLUSTER '{cluster}' SYNC;
