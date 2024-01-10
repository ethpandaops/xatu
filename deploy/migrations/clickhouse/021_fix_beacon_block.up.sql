ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_hash LowCardinality(String) AFTER meta_network_name,
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_next LowCardinality(String) AFTER meta_execution_fork_id_hash;

ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_hash LowCardinality(String) AFTER meta_network_name,
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_next LowCardinality(String) AFTER meta_execution_fork_id_hash;
