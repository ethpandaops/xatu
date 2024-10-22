CREATE TABLE default.canonical_execution_block_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_date_time` DateTime64(3) COMMENT 'The block timestamp' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'The block hash' CODEC(ZSTD(1)),
    `author` Nullable(String) COMMENT 'The block author' CODEC(ZSTD(1)),
    `gas_used` Nullable(UInt64) COMMENT 'The block gas used' CODEC(DoubleDelta, ZSTD(1)),
    `extra_data` Nullable(String) COMMENT 'The block extra data in hex' CODEC(ZSTD(1)),
    `extra_data_string` Nullable(String) COMMENT 'The block extra data in UTF-8 string' CODEC(ZSTD(1)),
    `base_fee_per_gas` Nullable(UInt64) COMMENT 'The block base fee per gas' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
    ) COMMENT 'Contains canonical execution block data.';

CREATE TABLE default.canonical_execution_block ON CLUSTER '{cluster}' AS default.canonical_execution_block_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_block_local,
    cityHash64(
        block_number,
        meta_network_name
    )
);

CREATE TABLE default.canonical_execution_transaction_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The transaction nonce' CODEC(ZSTD(1)),
    `from_address` String COMMENT 'The transaction from address' CODEC(ZSTD(1)),
    `to_address` Nullable(String) COMMENT 'The transaction to address' CODEC(ZSTD(1)),
    `value` UInt256 COMMENT 'The transaction value in float64' CODEC(ZSTD(1)),
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
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash
    ) COMMENT 'Contains canonical execution transaction data.';

CREATE TABLE default.canonical_execution_transaction ON CLUSTER '{cluster}' AS default.canonical_execution_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_transaction_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash
    )
);

CREATE TABLE default.canonical_execution_balance_diffs_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash that caused the balance diff' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the balance diff within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address of the balance diff' CODEC(ZSTD(1)),
    `from_value` UInt256 COMMENT 'The from value of the balance diff' CODEC(ZSTD(1)),
    `to_value` UInt256 COMMENT 'The to value of the balance diff' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution balance diff data.';

CREATE TABLE default.canonical_execution_balance_diffs ON CLUSTER '{cluster}' AS default.canonical_execution_balance_diffs_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_balance_diffs_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_balance_reads_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash that caused the balance read' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the balance read within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address of the balance read' CODEC(ZSTD(1)),
    `balance` UInt256 COMMENT 'The balance that was read' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution balance read data.';

CREATE TABLE default.canonical_execution_balance_reads ON CLUSTER '{cluster}' AS default.canonical_execution_balance_reads_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_balance_reads_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_four_byte_counts_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `signature` String COMMENT 'The signature of the four byte count' CODEC(ZSTD(1)),
    `size` UInt64 COMMENT 'The size of the four byte count' CODEC(ZSTD(1)),
    `count` UInt64 COMMENT 'The count of the four byte count' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash
    ) COMMENT 'Contains canonical execution four byte count data.';

CREATE TABLE default.canonical_execution_four_byte_counts ON CLUSTER '{cluster}' AS default.canonical_execution_four_byte_counts_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_four_byte_counts_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash
    )
);

CREATE TABLE default.canonical_execution_erc20_transfers_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the transfer within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `log_index` UInt64 COMMENT 'The log index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `erc20` String COMMENT 'The erc20 address' CODEC(ZSTD(1)),
    `from_address` String COMMENT 'The from address' CODEC(ZSTD(1)),
    `to_address` String COMMENT 'The to address' CODEC(ZSTD(1)),
    `value` UInt256 COMMENT 'The value of the transfer' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution erc20 transfer data.';

CREATE TABLE default.canonical_execution_erc20_transfers ON CLUSTER '{cluster}' AS default.canonical_execution_erc20_transfers_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_erc20_transfers_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_erc721_transfers_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the transfer within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `log_index` UInt64 COMMENT 'The log index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `erc20` String COMMENT 'The erc20 address' CODEC(ZSTD(1)),
    `from_address` String COMMENT 'The from address' CODEC(ZSTD(1)),
    `to_address` String COMMENT 'The to address' CODEC(ZSTD(1)),
    `token` UInt256 COMMENT 'The token id' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution erc721 transfer data.';

CREATE TABLE default.canonical_execution_erc721_transfers ON CLUSTER '{cluster}' AS default.canonical_execution_erc721_transfers_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_erc721_transfers_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_native_transfers_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the transfer within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `transfer_index` UInt64 COMMENT 'The transfer index' CODEC(DoubleDelta, ZSTD(1)),
    `from_address` String COMMENT 'The from address' CODEC(ZSTD(1)),
    `to_address` String COMMENT 'The to address' CODEC(ZSTD(1)),
    `value` UInt256 COMMENT 'The value of the approval' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution native transfer data.';

CREATE TABLE default.canonical_execution_native_transfers ON CLUSTER '{cluster}' AS default.canonical_execution_native_transfers_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_native_transfers_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_nonce_diffs_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash that caused the nonce diff' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the nonce diff within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address of the nonce diff' CODEC(ZSTD(1)),
    `from_value` UInt64 COMMENT 'The from value of the nonce diff' CODEC(ZSTD(1)),
    `to_value` UInt64 COMMENT 'The to value of the nonce diff' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution nonce diff data.';

CREATE TABLE default.canonical_execution_nonce_diffs ON CLUSTER '{cluster}' AS default.canonical_execution_nonce_diffs_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_nonce_diffs_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_nonce_reads_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash that caused the nonce read' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the nonce read within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address of the nonce read' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The nonce that was read' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution nonce read data.';

CREATE TABLE default.canonical_execution_nonce_reads ON CLUSTER '{cluster}' AS default.canonical_execution_nonce_reads_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_nonce_reads_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_address_appearances_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash that caused the address appearance' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the address appearance within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address of the address appearance' CODEC(ZSTD(1)),
    `relationship` LowCardinality(String) COMMENT 'The relationship of the address to the transaction',
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution address appearance data.';

CREATE TABLE default.canonical_execution_address_appearances ON CLUSTER '{cluster}' AS default.canonical_execution_address_appearances_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_address_appearances_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_contracts_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash that created the contract' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the contract creation within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `create_index` UInt32 COMMENT 'The create index' CODEC(DoubleDelta, ZSTD(1)),
    `contract_address` String COMMENT 'The contract address' CODEC(ZSTD(1)),
    `deployer` String COMMENT 'The address of the contract deployer' CODEC(ZSTD(1)),
    `factory` String COMMENT 'The address of the factory that deployed the contract' CODEC(ZSTD(1)),
    `init_code` String COMMENT 'The initialization code of the contract' CODEC(ZSTD(1)),
    `code` Nullable(String) COMMENT 'The code of the contract' CODEC(ZSTD(1)),
    `init_code_hash` String COMMENT 'The hash of the initialization code' CODEC(ZSTD(1)),
    `n_init_code_bytes` UInt32 COMMENT 'Number of bytes in the initialization code' CODEC(DoubleDelta, ZSTD(1)),
    `n_code_bytes` UInt32 COMMENT 'Number of bytes in the contract code' CODEC(DoubleDelta, ZSTD(1)),
    `code_hash` String COMMENT 'The hash of the contract code' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution contract data.';

CREATE TABLE default.canonical_execution_contracts ON CLUSTER '{cluster}' AS default.canonical_execution_contracts_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_contracts_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_traces_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt32 COMMENT 'The transaction index' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the trace within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `action_from` String COMMENT 'The from address of the action' CODEC(ZSTD(1)),
    `action_to` Nullable(String) COMMENT 'The to address of the action' CODEC(ZSTD(1)),
    `action_value` String COMMENT 'The value of the action' CODEC(ZSTD(1)),
    `action_gas` UInt32 COMMENT 'The gas provided for the action' CODEC(DoubleDelta, ZSTD(1)),
    `action_input` Nullable(String) COMMENT 'The input data for the action' CODEC(ZSTD(1)),
    `action_call_type` LowCardinality(String) COMMENT 'The call type of the action' CODEC(ZSTD(1)),
    `action_init` Nullable(String) COMMENT 'The initialization code for the action' CODEC(ZSTD(1)),
    `action_reward_type` String COMMENT 'The reward type for the action' CODEC(ZSTD(1)),
    `action_type` LowCardinality(String) COMMENT 'The type of the action' CODEC(ZSTD(1)),
    `result_gas_used` UInt32 COMMENT 'The gas used in the result' CODEC(DoubleDelta, ZSTD(1)),
    `result_output` Nullable(String) COMMENT 'The output of the result' CODEC(ZSTD(1)),
    `result_code` Nullable(String) COMMENT 'The code returned in the result' CODEC(ZSTD(1)),
    `result_address` Nullable(String) COMMENT 'The address returned in the result' CODEC(ZSTD(1)),
    `trace_address` Nullable(String) COMMENT 'The trace address' CODEC(ZSTD(1)),
    `subtraces` UInt32 COMMENT 'The number of subtraces' CODEC(DoubleDelta, ZSTD(1)),
    `error` Nullable(String) COMMENT 'The error, if any, in the trace' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution traces data.';

CREATE TABLE default.canonical_execution_traces ON CLUSTER '{cluster}' AS default.canonical_execution_traces_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_traces_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_logs_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt32 COMMENT 'The transaction index' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash associated with the log' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the log within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `log_index` UInt32 COMMENT 'The log index within the block' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address associated with the log' CODEC(ZSTD(1)),
    `topic0` String COMMENT 'The first topic of the log' CODEC(ZSTD(1)),
    `topic1` Nullable(String) COMMENT 'The second topic of the log' CODEC(ZSTD(1)),
    `topic2` Nullable(String) COMMENT 'The third topic of the log' CODEC(ZSTD(1)),
    `topic3` Nullable(String) COMMENT 'The fourth topic of the log' CODEC(ZSTD(1)),
    `data` Nullable(String) COMMENT 'The data associated with the log' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution logs data.';

CREATE TABLE default.canonical_execution_logs ON CLUSTER '{cluster}' AS default.canonical_execution_logs_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_logs_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_storage_diffs_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt32 COMMENT 'The transaction index' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash associated with the storage diff' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the storage diff within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `address` String COMMENT 'The address associated with the storage diff' CODEC(ZSTD(1)),
    `slot` String COMMENT 'The storage slot key' CODEC(ZSTD(1)),
    `from_value` String COMMENT 'The original value before the storage diff' CODEC(ZSTD(1)),
    `to_value` String COMMENT 'The new value after the storage diff' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution storage diffs data.';

CREATE TABLE default.canonical_execution_storage_diffs ON CLUSTER '{cluster}' AS default.canonical_execution_storage_diffs_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_storage_diffs_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);

CREATE TABLE default.canonical_execution_storage_reads_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt32 COMMENT 'The transaction index' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash associated with the storage read' CODEC(ZSTD(1)),
    `internal_index` UInt32 COMMENT 'The internal index of the storage read within the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `contract_address` String COMMENT 'The contract address associated with the storage read' CODEC(ZSTD(1)),
    `slot` String COMMENT 'The storage slot key' CODEC(ZSTD(1)),
    `value` String COMMENT 'The value read from the storage slot' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY
    (
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    ) COMMENT 'Contains canonical execution storage reads data.';

CREATE TABLE default.canonical_execution_storage_reads ON CLUSTER '{cluster}' AS default.canonical_execution_storage_reads_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_execution_storage_reads_local,
    cityHash64(
        block_number,
        meta_network_name,
        transaction_hash,
        internal_index
    )
);
