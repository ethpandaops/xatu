ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN meta_execution_fork_id_hash,
    DROP COLUMN meta_execution_fork_id_next;

ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN meta_execution_fork_id_hash,
    DROP COLUMN meta_execution_fork_id_next;
