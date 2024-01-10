ALTER TABLE beacon_api_eth_v2_beacon_block on cluster '{cluster}'
    DROP COLUMN block_total_bytes,
    DROP COLUMN block_total_bytes_compressed,
    DROP COLUMN execution_payload_transactions_total_bytes_compressed;

ALTER TABLE beacon_api_eth_v2_beacon_block_local on cluster '{cluster}'
    DROP COLUMN block_total_bytes,
    DROP COLUMN block_total_bytes_compressed,
    DROP COLUMN execution_payload_transactions_total_bytes_compressed;
