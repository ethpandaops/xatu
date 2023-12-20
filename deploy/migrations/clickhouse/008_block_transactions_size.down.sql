ALTER TABLE beacon_api_eth_v2_beacon_block on cluster '{cluster}'
    DROP COLUMN execution_payload_transactions_count,
    DROP COLUMN execution_payload_transactions_total_bytes;

ALTER TABLE beacon_api_eth_v2_beacon_block_local on cluster '{cluster}'
    DROP COLUMN execution_payload_transactions_count,
    DROP COLUMN execution_payload_transactions_total_bytes;
