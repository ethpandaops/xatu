CREATE TABLE default.execution_transaction_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'The block hash' CODEC(ZSTD(1)),
    `parent_hash` FixedString(66) COMMENT 'The parent block hash' CODEC(ZSTD(1)),
    `position` UInt32 COMMENT 'The position of the transaction in the beacon block' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'The hash of the transaction' CODEC(ZSTD(1)),
    `from` FixedString(42) COMMENT 'The address of the account that sent the transaction' CODEC(ZSTD(1)),
    `to` Nullable(FixedString(42)) COMMENT 'The address of the account that is the transaction recipient' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The nonce of the sender account at the time of the transaction' CODEC(ZSTD(1)),
    `gas_price` UInt128 COMMENT 'The gas price of the transaction in wei' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'The maximum gas provided for the transaction execution' CODEC(ZSTD(1)),
    `gas_tip_cap` Nullable(UInt128) COMMENT 'The priority fee (tip) the user has set for the transaction' CODEC(ZSTD(1)),
    `gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee the user has set for the transaction' CODEC(ZSTD(1)),
    `value` UInt128 COMMENT 'The value transferred with the transaction in wei' CODEC(ZSTD(1)),
    `type` UInt8 COMMENT 'The type of the transaction' CODEC(ZSTD(1)),
    `size` UInt32 COMMENT 'The size of the transaction data in bytes' CODEC(ZSTD(1)),
    `call_data_size` UInt32 COMMENT 'The size of the call data of the transaction in bytes' CODEC(ZSTD(1)),
    `blob_gas` Nullable(UInt64) COMMENT 'The maximum gas provided for the blob transaction execution' CODEC(ZSTD(1)),
    `blob_gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee the user has set for the transaction' CODEC(ZSTD(1)),
    `blob_hashes` Array(String) COMMENT 'The hashes of the blob commitments for blob transactions' CODEC(ZSTD(1)),
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
        block_number,
        meta_network_name,
        block_hash,
        position
    ) COMMENT 'Contains execution transaction data that may not be canonical.';

CREATE TABLE default.execution_transaction ON CLUSTER '{cluster}' AS default.execution_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    execution_transaction_local,
    cityHash64(
        block_number,
        meta_network_name,
        block_hash,
        position
    )
);
