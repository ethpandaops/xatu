ALTER TABLE canonical_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN execution_payload_base_fee_per_gas Nullable(UInt128) COMMENT 'Base fee per gas for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_fee_recipient,
    ADD COLUMN execution_payload_blob_gas_used Nullable(UInt64) COMMENT 'Gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_base_fee_per_gas,
    ADD COLUMN execution_payload_excess_blob_gas Nullable(UInt64) COMMENT 'Excess gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_blob_gas_used,
    ADD COLUMN execution_payload_gas_limit Nullable(UInt64) COMMENT 'Gas limit for execution payload' CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas,
    ADD COLUMN execution_payload_gas_used Nullable(UInt64) COMMENT 'Gas used for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_gas_limit;

ALTER TABLE canonical_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN execution_payload_base_fee_per_gas Nullable(UInt128) COMMENT 'Base fee per gas for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_fee_recipient,
    ADD COLUMN execution_payload_blob_gas_used Nullable(UInt64) COMMENT 'Gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_base_fee_per_gas,
    ADD COLUMN execution_payload_excess_blob_gas Nullable(UInt64) COMMENT 'Excess gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_blob_gas_used,
    ADD COLUMN execution_payload_gas_limit Nullable(UInt64) COMMENT 'Gas limit for execution payload' CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas,
    ADD COLUMN execution_payload_gas_used Nullable(UInt64) COMMENT 'Gas used for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_gas_limit;

ALTER TABLE beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN execution_payload_base_fee_per_gas Nullable(UInt128) COMMENT 'Base fee per gas for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_fee_recipient,
    ADD COLUMN execution_payload_blob_gas_used Nullable(UInt64) COMMENT 'Gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_base_fee_per_gas,
    ADD COLUMN execution_payload_excess_blob_gas Nullable(UInt64) COMMENT 'Excess gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_blob_gas_used,
    ADD COLUMN execution_payload_gas_limit Nullable(UInt64) COMMENT 'Gas limit for execution payload' CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas,
    ADD COLUMN execution_payload_gas_used Nullable(UInt64) COMMENT 'Gas used for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_gas_limit;

ALTER TABLE beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN execution_payload_base_fee_per_gas Nullable(UInt128) COMMENT 'Base fee per gas for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_fee_recipient,
    ADD COLUMN execution_payload_blob_gas_used Nullable(UInt64) COMMENT 'Gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_base_fee_per_gas,
    ADD COLUMN execution_payload_excess_blob_gas Nullable(UInt64) COMMENT 'Excess gas used for blobs in execution payload' CODEC(ZSTD(1)) AFTER execution_payload_blob_gas_used,
    ADD COLUMN execution_payload_gas_limit Nullable(UInt64) COMMENT 'Gas limit for execution payload' CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas,
    ADD COLUMN execution_payload_gas_used Nullable(UInt64) COMMENT 'Gas used for execution payload' CODEC(ZSTD(1)) AFTER execution_payload_gas_limit;
