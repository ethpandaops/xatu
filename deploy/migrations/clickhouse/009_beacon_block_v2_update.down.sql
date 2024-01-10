ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS meta_consensus_version,
    DROP COLUMN IF EXISTS meta_consensus_version_major,
    DROP COLUMN IF EXISTS meta_consensus_version_minor,
    DROP COLUMN IF EXISTS meta_consensus_version_patch,
    DROP COLUMN IF EXISTS meta_consensus_implementation,
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_hash LowCardinality(String) AFTER meta_network_name,
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_next LowCardinality(String) AFTER meta_execution_fork_id_hash,
    COMMENT COLUMN execution_payload_transactions_count '',
    COMMENT COLUMN execution_payload_transactions_total_bytes '';

ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS meta_consensus_version,
    DROP COLUMN IF EXISTS meta_consensus_version_major,
    DROP COLUMN IF EXISTS meta_consensus_version_minor,
    DROP COLUMN IF EXISTS meta_consensus_version_patch,
    DROP COLUMN IF EXISTS meta_consensus_implementation,
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_hash LowCardinality(String) AFTER meta_network_name,
    ADD COLUMN IF NOT EXISTS meta_execution_fork_id_next LowCardinality(String) AFTER meta_execution_fork_id_hash,
    COMMENT COLUMN execution_payload_transactions_count '',
    COMMENT COLUMN execution_payload_transactions_total_bytes '';
