CREATE TABLE default.execution_transaction_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'The block hash' CODEC(ZSTD(1)),
    `parent_hash` FixedString(66) COMMENT 'The parent block hash' CODEC(ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The transaction nonce' CODEC(ZSTD(1)),
    `from_address` String COMMENT 'The transaction from address' CODEC(ZSTD(1)),
    `to_address` Nullable(String) COMMENT 'The transaction to address' CODEC(ZSTD(1)),
    `value` UInt256 COMMENT 'The transaction value in wei' CODEC(ZSTD(1)),
    `input` Nullable(String) COMMENT 'The transaction input in hex' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'The transaction gas limit' CODEC(ZSTD(1)),
    `gas_used` UInt64 COMMENT 'The transaction gas used' CODEC(ZSTD(1)),
    `gas_price` UInt64 COMMENT 'The transaction gas price' CODEC(ZSTD(1)),
    `transaction_type` UInt32 COMMENT 'The transaction type' CODEC(ZSTD(1)),
    `max_priority_fee_per_gas` UInt64 COMMENT 'The transaction max priority fee per gas' CODEC(ZSTD(1)),
    `max_fee_per_gas` UInt64 COMMENT 'The transaction max fee per gas' CODEC(ZSTD(1)),
    `success` Bool COMMENT 'The transaction success' CODEC(ZSTD(1)),
    `n_input_bytes` UInt32 COMMENT 'The transaction input bytes' CODEC(ZSTD(1)),
    `n_input_zero_bytes` UInt32 COMMENT 'The transaction input zero bytes' CODEC(ZSTD(1)),
    `n_input_nonzero_bytes` UInt32 COMMENT 'The transaction input nonzero bytes' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        meta_network_name,
        block_number,
        block_hash,
        transaction_index
    ) COMMENT 'Contains execution transaction data that may not be canonical.';

CREATE TABLE default.execution_transaction ON CLUSTER '{cluster}' AS default.execution_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    execution_transaction_local,
    cityHash64(
        block_number,
        meta_network_name,
        block_hash,
        transaction_hash
    )
);
