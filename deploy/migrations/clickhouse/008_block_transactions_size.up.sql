ALTER TABLE beacon_api_eth_v2_beacon_block_local on cluster '{cluster}'
    ADD COLUMN execution_payload_transactions_count Nullable(UInt32) Codec(ZSTD(1)) AFTER execution_payload_parent_hash,
    ADD COLUMN execution_payload_transactions_total_bytes Nullable(UInt32) Codec(ZSTD(1)) AFTER execution_payload_transactions_count;

ALTER TABLE beacon_api_eth_v2_beacon_block on cluster '{cluster}'
    ADD COLUMN execution_payload_transactions_count Nullable(UInt32) Codec(ZSTD(1)) AFTER execution_payload_parent_hash,
    ADD COLUMN execution_payload_transactions_total_bytes Nullable(UInt32) Codec(ZSTD(1)) AFTER execution_payload_transactions_count;
