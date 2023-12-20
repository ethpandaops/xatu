DROP TABLE IF EXISTS canonical_beacon_block on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_proposer_slashing on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_proposer_slashing_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_attester_slashing on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_attester_slashing_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_bls_to_execution_change on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_bls_to_execution_change_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_transaction on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_transaction_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_voluntary_exit on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_voluntary_exit_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_deposit on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_deposit_local on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_withdrawal on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_withdrawal_local on cluster '{cluster}' SYNC;

ALTER TABLE beacon_api_eth_v2_beacon_block_local on cluster '{cluster}'
    DROP COLUMN block_version;

ALTER TABLE beacon_api_eth_v2_beacon_block on cluster '{cluster}'
    DROP COLUMN block_version;
