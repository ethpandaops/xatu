ALTER TABLE beacon_api_eth_v2_beacon_block_local on cluster '{cluster}'
    ADD COLUMN block_total_bytes Nullable(UInt32) Codec(ZSTD(1)) AFTER block_root,
    ADD COLUMN block_total_bytes_compressed Nullable(UInt32) Codec(ZSTD(1)) AFTER block_total_bytes,
    ADD COLUMN execution_payload_transactions_total_bytes_compressed Nullable(UInt32) Codec(ZSTD(1)) AFTER execution_payload_transactions_total_bytes,
    COMMENT COLUMN block_total_bytes 'The total bytes of the beacon block payload',
    COMMENT COLUMN block_total_bytes_compressed 'The total bytes of the beacon block payload when compressed using snappy',
    COMMENT COLUMN execution_payload_transactions_total_bytes_compressed 'The transaction total bytes of the execution payload when compressed using snappy';

ALTER TABLE beacon_api_eth_v2_beacon_block on cluster '{cluster}'
    ADD COLUMN block_total_bytes Nullable(UInt32) Codec(ZSTD(1)) AFTER block_root,
    ADD COLUMN block_total_bytes_compressed Nullable(UInt32) Codec(ZSTD(1)) AFTER block_total_bytes,
    ADD COLUMN execution_payload_transactions_total_bytes_compressed Nullable(UInt32) Codec(ZSTD(1)) AFTER execution_payload_transactions_total_bytes,
    COMMENT COLUMN block_total_bytes 'The total bytes of the beacon block payload',
    COMMENT COLUMN block_total_bytes_compressed 'The total bytes of the beacon block payload when compressed using snappy',
    COMMENT COLUMN execution_payload_transactions_total_bytes_compressed 'The transaction total bytes of the execution payload when compressed using snappy';
