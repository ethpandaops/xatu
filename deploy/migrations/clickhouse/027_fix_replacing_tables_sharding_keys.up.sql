DROP TABLE IF EXISTS canonical_beacon_block on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block on cluster '{cluster}' AS canonical_beacon_block_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_local, cityHash64(slot_start_date_time, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_proposer_slashing on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_proposer_slashing on cluster '{cluster}' AS canonical_beacon_block_proposer_slashing_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_proposer_slashing_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_attester_slashing on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_attester_slashing on cluster '{cluster}' AS canonical_beacon_block_attester_slashing_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_attester_slashing_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_bls_to_execution_change on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_bls_to_execution_change on cluster '{cluster}' AS canonical_beacon_block_bls_to_execution_change_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_bls_to_execution_change_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_execution_transaction on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_execution_transaction on cluster '{cluster}' AS canonical_beacon_block_execution_transaction_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_execution_transaction_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_voluntary_exit on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_voluntary_exit on cluster '{cluster}' AS canonical_beacon_block_voluntary_exit_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_voluntary_exit_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_deposit on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_deposit on cluster '{cluster}' AS canonical_beacon_block_deposit_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_deposit_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_block_withdrawal on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_block_withdrawal on cluster '{cluster}' AS canonical_beacon_block_withdrawal_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_block_withdrawal_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS beacon_block_classification on cluster '{cluster}' SYNC;
CREATE TABLE beacon_block_classification on cluster '{cluster}' AS beacon_block_classification_local
ENGINE = Distributed('{cluster}', default, beacon_block_classification_local, cityHash64(slot_start_date_time, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_blob_sidecar on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_blob_sidecar on cluster '{cluster}' AS canonical_beacon_blob_sidecar_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_blob_sidecar_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS mempool_dumpster_transaction on cluster '{cluster}' SYNC;
CREATE TABLE mempool_dumpster_transaction on cluster '{cluster}' AS mempool_dumpster_transaction_local
ENGINE = Distributed('{cluster}', default, mempool_dumpster_transaction_local, cityHash64(timestamp, unique_key, chain_id));

DROP TABLE IF EXISTS block_native_mempool_transaction on cluster '{cluster}' SYNC;
CREATE TABLE block_native_mempool_transaction on cluster '{cluster}' AS block_native_mempool_transaction_local
ENGINE = Distributed('{cluster}', default, block_native_mempool_transaction_local, cityHash64(detecttime, unique_key, network));

DROP TABLE IF EXISTS beacon_p2p_attestation on cluster '{cluster}' SYNC;
CREATE TABLE beacon_p2p_attestation on cluster '{cluster}' AS beacon_p2p_attestation_local
ENGINE = Distributed('{cluster}', default, beacon_p2p_attestation_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_proposer_duty on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_proposer_duty on cluster '{cluster}' AS canonical_beacon_proposer_duty_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_proposer_duty_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

DROP TABLE IF EXISTS canonical_beacon_elaborated_attestation on cluster '{cluster}' SYNC;
CREATE TABLE canonical_beacon_elaborated_attestation on cluster '{cluster}' AS canonical_beacon_elaborated_attestation_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_elaborated_attestation_local, cityHash64(slot_start_date_time, unique_key, meta_network_name));

