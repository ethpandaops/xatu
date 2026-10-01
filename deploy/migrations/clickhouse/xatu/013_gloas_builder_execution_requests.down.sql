-- Drop cannon Gloas (EIP-8282) builder execution request tables.

-- builder_exit
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_builder_exit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_builder_exit_local ON CLUSTER '{cluster}' SYNC;

-- builder_deposit
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_builder_deposit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_block_execution_request_builder_deposit_local ON CLUSTER '{cluster}' SYNC;
