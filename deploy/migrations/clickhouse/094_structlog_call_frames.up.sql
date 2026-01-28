ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN gas_self UInt64 DEFAULT 0 COMMENT 'Gas consumed by this opcode only, excludes child frame gas for CALL/CREATE opcodes. sum(gas_self) = total execution gas without double counting' CODEC(ZSTD(1)) AFTER gas_used;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN gas_self UInt64 DEFAULT 0 COMMENT 'Gas consumed by this opcode only, excludes child frame gas for CALL/CREATE opcodes. sum(gas_self) = total execution gas without double counting' CODEC(ZSTD(1)) AFTER gas_used;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_id UInt32 DEFAULT 0 COMMENT 'Sequential identifier for the call frame within the transaction' CODEC(DoubleDelta, ZSTD(1)) AFTER call_to_address;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_id UInt32 DEFAULT 0 COMMENT 'Sequential identifier for the call frame within the transaction' CODEC(DoubleDelta, ZSTD(1)) AFTER call_to_address;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_path Array(UInt32) DEFAULT [0] COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)) AFTER call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_path Array(UInt32) DEFAULT [0] COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)) AFTER call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN program_counter;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN program_counter;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN meta_network_id;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN meta_network_id;

ALTER TABLE admin.execution_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS complete UInt8;

ALTER TABLE admin.execution_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS complete UInt8;

ALTER TABLE admin.execution_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS task_count UInt32;

ALTER TABLE admin.execution_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS task_count UInt32;

-- Create canonical_execution_transaction_structlog_agg table for aggregated structlog data
-- This table stores per-frame summary rows (operation='') and per-opcode aggregated rows
-- Produces ~50-200x fewer rows than raw structlog table
CREATE TABLE default.canonical_execution_transaction_structlog_agg_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `transaction_index` UInt32 COMMENT 'The transaction position in the block' CODEC(DoubleDelta, ZSTD(1)),
    `call_frame_id` UInt32 COMMENT 'Sequential frame ID within the transaction (0=root)' CODEC(DoubleDelta, ZSTD(1)),
    `parent_call_frame_id` Nullable(UInt32) COMMENT 'Parent frame ID (NULL for root frame)' CODEC(ZSTD(1)),
    `call_frame_path` Array(UInt32) COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)),
    `depth` UInt32 COMMENT 'Call nesting depth (0=root)' CODEC(DoubleDelta, ZSTD(1)),
    `target_address` Nullable(String) COMMENT 'Contract address being called' CODEC(ZSTD(1)),
    `call_type` LowCardinality(String) COMMENT 'Call type: CALL/DELEGATECALL/STATICCALL/CALLCODE/CREATE/CREATE2 (empty for root)',
    `operation` LowCardinality(String) COMMENT 'Opcode name for per-opcode rows, empty string for frame summary rows',
    `opcode_count` UInt64 COMMENT 'Number of opcodes (total for summary row, count for per-opcode row)' CODEC(ZSTD(1)),
    `error_count` UInt64 COMMENT 'Number of errors' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'Gas consumed: SUM(gas_self) for per-opcode, frame self gas for summary' CODEC(ZSTD(1)),
    `gas_cumulative` UInt64 COMMENT 'Cumulative gas: SUM(gas_used) for per-opcode, frame total for summary' CODEC(ZSTD(1)),
    `min_depth` UInt32 COMMENT 'Minimum depth where opcode appeared (per-opcode rows)' CODEC(DoubleDelta, ZSTD(1)),
    `max_depth` UInt32 COMMENT 'Maximum depth where opcode appeared (per-opcode rows)' CODEC(DoubleDelta, ZSTD(1)),
    `gas_refund` Nullable(UInt64) COMMENT 'Gas refund (root summary row only)' CODEC(ZSTD(1)),
    `intrinsic_gas` Nullable(UInt64) COMMENT 'Intrinsic gas (root summary row only, computed)' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 201600) -- roughly 1 month of blocks
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        call_frame_id,
        operation
    ) COMMENT 'Aggregated EVM execution data. Summary rows (operation="") contain frame metadata. Per-opcode rows contain aggregated gas/count per (frame, opcode).';

CREATE TABLE default.canonical_execution_transaction_structlog_agg ON CLUSTER '{cluster}' AS default.canonical_execution_transaction_structlog_agg_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_transaction_structlog_agg_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        call_frame_id
    )
);
