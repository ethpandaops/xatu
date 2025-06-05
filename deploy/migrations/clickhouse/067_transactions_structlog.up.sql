CREATE DATABASE IF NOT EXISTS `admin` ON CLUSTER '{cluster}';

CREATE TABLE admin.execution_block_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime(3) CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `processor` LowCardinality(String) COMMENT 'The type of processor that processed the block',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
)
ORDER BY
    (
        block_number,
        processor,
        meta_network_name
    );

CREATE TABLE admin.execution_block ON CLUSTER '{cluster}' AS admin.execution_block_local ENGINE = Distributed(
    '{cluster}',
    admin,
    execution_block_local,
    cityHash64(
        block_number,
        processor,
        meta_network_name
    )
);

CREATE TABLE default.canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `transaction_index` UInt32 COMMENT 'The transaction position in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_gas` UInt64 COMMENT 'The transaction gas' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_failed` Bool COMMENT 'The transaction failed' CODEC(ZSTD(1)),
    `transaction_return_value` Nullable(String) COMMENT 'The transaction return value' CODEC(ZSTD(1)),
    `index` UInt32 COMMENT 'The index of this structlog in this transaction' CODEC(DoubleDelta, ZSTD(1)),
    `program_counter` UInt32 COMMENT 'The program counter' CODEC(Delta, ZSTD(1)),
    `operation` LowCardinality(String) COMMENT 'The operation',
    `gas` UInt64 COMMENT 'The gas' CODEC(Delta, ZSTD(1)),
    `gas_cost` UInt64 COMMENT 'The gas cost' CODEC(DoubleDelta, ZSTD(1)),
    `depth` UInt64 COMMENT 'The depth' CODEC(DoubleDelta, ZSTD(1)),
    `return_data` Nullable(String) COMMENT 'The return data' CODEC(ZSTD(1)),
    `refund` Nullable(UInt64) COMMENT 'The refund' CODEC(ZSTD(1)),
    `error` Nullable(String) COMMENT 'The error' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
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
        `index`
    ) COMMENT 'Contains canonical execution transaction structlog data.';

CREATE TABLE default.canonical_execution_transaction_structlog ON CLUSTER '{cluster}' AS default.canonical_execution_transaction_structlog_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_transaction_structlog_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        `index`
    )
);