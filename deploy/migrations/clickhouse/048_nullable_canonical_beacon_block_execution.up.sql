ALTER TABLE default.canonical_beacon_block_local ON CLUSTER '{cluster}'
    MODIFY COLUMN `execution_payload_block_hash` Nullable(FixedString(66)) COMMENT 'The block hash of the execution payload' CODEC(ZSTD(1)),
    MODIFY COLUMN `execution_payload_block_number` Nullable(UInt32) COMMENT 'The block number of the execution payload' CODEC(DoubleDelta, ZSTD(1)),
    MODIFY COLUMN `execution_payload_fee_recipient` Nullable(String) COMMENT 'The recipient of the fee for this execution payload' CODEC(ZSTD(1)),
    MODIFY COLUMN `execution_payload_state_root` Nullable(FixedString(66)) COMMENT 'The state root of the execution payload' CODEC(ZSTD(1)),
    MODIFY COLUMN `execution_payload_parent_hash` Nullable(FixedString(66)) COMMENT 'The parent hash of the execution payload' CODEC(ZSTD(1));

ALTER TABLE default.canonical_beacon_block ON CLUSTER '{cluster}'
    MODIFY COLUMN `execution_payload_block_hash` Nullable(FixedString(66)) COMMENT 'The block hash of the execution payload' CODEC(ZSTD(1)),
    MODIFY COLUMN `execution_payload_block_number` Nullable(UInt32) COMMENT 'The block number of the execution payload' CODEC(DoubleDelta, ZSTD(1)),
    MODIFY COLUMN `execution_payload_fee_recipient` Nullable(String) COMMENT 'The recipient of the fee for this execution payload' CODEC(ZSTD(1)),
    MODIFY COLUMN `execution_payload_state_root` Nullable(FixedString(66)) COMMENT 'The state root of the execution payload' CODEC(ZSTD(1)),
    MODIFY COLUMN `execution_payload_parent_hash` Nullable(FixedString(66)) COMMENT 'The parent hash of the execution payload' CODEC(ZSTD(1));
