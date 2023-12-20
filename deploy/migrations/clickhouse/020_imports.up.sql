CREATE TABLE default.imported_sources_local on cluster '{cluster}'
(
  create_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  target_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  source LowCardinality(String)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(create_date_time)
ORDER BY (create_date_time, source);

ALTER TABLE default.imported_sources_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'This table contains the list of sources that have been imported into the database',
COMMENT COLUMN create_date_time 'Creation date of this row',
COMMENT COLUMN target_date_time 'The date of the data that was imported',
COMMENT COLUMN source 'Source of the data that was imported';

CREATE TABLE imported_sources on cluster '{cluster}' AS imported_sources_local
ENGINE = Distributed('{cluster}', default, imported_sources_local, rand());

CREATE TABLE default.mempool_dumpster_transaction_local on cluster '{cluster}'
(
  unique_key Int64 CODEC(ZSTD(1)),
  updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
  timestamp_ms DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  hash FixedString(66) Codec(ZSTD(1)),
  chain_id UInt32 Codec(ZSTD(1)),
  from FixedString(42) Codec(ZSTD(1)),
  to Nullable(FixedString(42)) Codec(ZSTD(1)),
  value UInt128 Codec(ZSTD(1)),
  nonce UInt64 Codec(ZSTD(1)),
  gas UInt64 Codec(ZSTD(1)),
  gas_price UInt128 Codec(ZSTD(1)),
  gas_tip_cap Nullable(UInt128) Codec(ZSTD(1)),
  gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)),
  data_size UInt32 Codec(ZSTD(1)),
  data_4bytes Nullable(FixedString(10)) Codec(ZSTD(1)),
  sources Array(LowCardinality(String)),
  included_at_block_height Nullable(UInt64) Codec(ZSTD(1)),
  included_block_timestamp_ms Nullable(DateTime64(3)) Codec(DoubleDelta, ZSTD(1)),
  inclusion_delay_ms Nullable(Int64) Codec(ZSTD(1))
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(timestamp_ms)
ORDER BY (timestamp_ms, unique_key, chain_id)
TTL toDateTime(timestamp_ms) TO VOLUME 'default',
    toDateTime(timestamp_ms) + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    toDateTime(timestamp_ms) + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    toDateTime(timestamp_ms) + INTERVAL 40 MONTH DELETE;

ALTER TABLE default.mempool_dumpster_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from mempool dumpster dataset',
COMMENT COLUMN unique_key 'Unique key for the row, this is outside the source data and used for deduplication',
COMMENT COLUMN updated_date_time 'When this row was last updated, this is outside the source data and used for deduplication',
COMMENT COLUMN timestamp_ms 'Timestamp of the transaction',
COMMENT COLUMN hash 'The hash of the transaction',
COMMENT COLUMN chain_id 'The chain id of the transaction',
COMMENT COLUMN from 'The address of the account that sent the transaction',
COMMENT COLUMN to 'The address of the account that is the transaction recipient',
COMMENT COLUMN value 'The value transferred with the transaction in wei',
COMMENT COLUMN nonce 'The nonce of the sender account at the time of the transaction',
COMMENT COLUMN gas 'The maximum gas provided for the transaction execution',
COMMENT COLUMN gas_price 'The gas price of the transaction in wei',
COMMENT COLUMN gas_tip_cap 'The gas tip cap of the transaction in wei',
COMMENT COLUMN gas_fee_cap 'The gas fee cap of the transaction in wei',
COMMENT COLUMN data_size 'The size of the call data of the transaction in bytes',
COMMENT COLUMN data_4bytes 'The first 4 bytes of the call data of the transaction',
COMMENT COLUMN sources 'The sources that saw this transaction in their mempool',
COMMENT COLUMN included_at_block_height 'The block height at which this transaction was included',
COMMENT COLUMN included_block_timestamp_ms 'The timestamp of the block at which this transaction was included',
COMMENT COLUMN inclusion_delay_ms 'The delay between the transaction timestamp and the block timestamp';

CREATE TABLE mempool_dumpster_transaction on cluster '{cluster}' AS mempool_dumpster_transaction_local
ENGINE = Distributed('{cluster}', default, mempool_dumpster_transaction_local, rand());

CREATE TABLE default.mempool_dumpster_transaction_source_local on cluster '{cluster}'
(
  unique_key Int64 CODEC(ZSTD(1)),
  updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
  timestamp_ms DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  hash FixedString(66) Codec(ZSTD(1)),
  source LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(timestamp_ms)
ORDER BY (timestamp_ms, unique_key)
TTL toDateTime(timestamp_ms) TO VOLUME 'default',
    toDateTime(timestamp_ms) + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    toDateTime(timestamp_ms) + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    toDateTime(timestamp_ms) + INTERVAL 40 MONTH DELETE;

ALTER TABLE default.mempool_dumpster_transaction_source_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from mempool dumpster dataset',
COMMENT COLUMN unique_key 'Unique key for the row, this is outside the source data and used for deduplication',
COMMENT COLUMN updated_date_time 'When this row was last updated, this is outside the source data and used for deduplication',
COMMENT COLUMN timestamp_ms 'Timestamp of the transaction',
COMMENT COLUMN hash 'The hash of the transaction',
COMMENT COLUMN source 'The source that saw this transaction in their mempool';

CREATE TABLE mempool_dumpster_transaction_source on cluster '{cluster}' AS mempool_dumpster_transaction_source_local
ENGINE = Distributed('{cluster}', default, mempool_dumpster_transaction_source_local, rand());

CREATE TABLE default.block_native_mempool_transaction_local on cluster '{cluster}'
(
  unique_key Int64 CODEC(ZSTD(1)),
  updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
  detecttime DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  hash FixedString(66) Codec(ZSTD(1)),
  status LowCardinality(String),
  region LowCardinality(String),
  reorg Nullable(FixedString(66)) Codec(ZSTD(1)),
  replace Nullable(FixedString(66)) Codec(ZSTD(1)),
  curblocknumber Nullable(UInt64) Codec(ZSTD(1)),
  failurereason Nullable(String) Codec(ZSTD(1)),
  blockspending Nullable(UInt64) Codec(ZSTD(1)),
  timepending Nullable(UInt64) Codec(ZSTD(1)),
  nonce UInt64 Codec(ZSTD(1)),
  gas UInt64 Codec(ZSTD(1)),
  gasprice UInt128 Codec(ZSTD(1)),
  value UInt128 Codec(ZSTD(1)),
  toaddress Nullable(FixedString(42)) Codec(ZSTD(1)),
  fromaddress FixedString(42) Codec(ZSTD(1)),
  datasize UInt32 Codec(ZSTD(1)),
  data4bytes Nullable(FixedString(10)) Codec(ZSTD(1)),
  network LowCardinality(String),
  type UInt8 Codec(ZSTD(1)),
  maxpriorityfeepergas Nullable(UInt128) Codec(ZSTD(1)),
  maxfeepergas Nullable(UInt128) Codec(ZSTD(1)),
  basefeepergas Nullable(UInt128) Codec(ZSTD(1)),
  dropreason Nullable(String) Codec(ZSTD(1)),
  rejectionreason Nullable(String) Codec(ZSTD(1)),
  stuck Bool Codec(ZSTD(1)),
  gasused Nullable(UInt64) Codec(ZSTD(1))
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(detecttime)
ORDER BY (detecttime, unique_key, network)
TTL toDateTime(detecttime) TO VOLUME 'default',
    toDateTime(detecttime) + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    toDateTime(detecttime) + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    toDateTime(detecttime) + INTERVAL 40 MONTH DELETE;

ALTER TABLE default.block_native_mempool_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from block native mempool dataset',
COMMENT COLUMN unique_key 'Unique key for the row, this is outside the source data and used for deduplication',
COMMENT COLUMN updated_date_time 'When this row was last updated, this is outside the source data and used for deduplication',
COMMENT COLUMN detecttime 'Timestamp that the transaction was detected in mempool',
COMMENT COLUMN hash 'Unique identifier hash for a given transaction',
COMMENT COLUMN status 'Status of the transaction',
COMMENT COLUMN region 'The geographic region for the node that detected the transaction',
COMMENT COLUMN reorg 'If there was a reorg, refers to the blockhash of the reorg',
COMMENT COLUMN replace 'If the transaction was replaced (speedup/cancel), the transaction hash of the replacement',
COMMENT COLUMN curblocknumber 'The block number the event was detected in',
COMMENT COLUMN failurereason 'If a transaction failed, this field provides contextual information',
COMMENT COLUMN blockspending 'If a transaction was finalized (confirmed, failed), this refers to the number of blocks that the transaction was waiting to get on-chain',
COMMENT COLUMN timepending 'If a transaction was finalized (confirmed, failed), this refers to the time in milliseconds that the transaction was waiting to get on-chain',
COMMENT COLUMN nonce 'A unique number which counts the number of transactions sent from a given address',
COMMENT COLUMN gas 'The maximum number of gas units allowed for the transaction',
COMMENT COLUMN gasprice 'The price offered to the miner/validator per unit of gas. Denominated in wei',
COMMENT COLUMN value 'The amount of ETH transferred or sent to contract. Denominated in wei',
COMMENT COLUMN toaddress 'The destination of a given transaction',
COMMENT COLUMN fromaddress 'The source/initiator of a given transaction',
COMMENT COLUMN datasize 'The size of the call data of the transaction in bytes',
COMMENT COLUMN data4bytes 'The first 4 bytes of the call data of the transaction',
COMMENT COLUMN network 'The specific Ethereum network used',
COMMENT COLUMN type 'Post EIP-1559, this indicates how the gas parameters are submitted to the network: - type 0 - legacy - type 1 - usage of access lists according to EIP-2930 - type 2 - using maxpriorityfeepergas and maxfeepergas',
COMMENT COLUMN maxpriorityfeepergas 'The maximum value for a tip offered to the miner/validator per unit of gas. The actual tip paid can be lower if (maxfee - basefee) < maxpriorityfee. Denominated in wei',
COMMENT COLUMN maxfeepergas 'The maximum value for the transaction fee (including basefee and tip) offered to the miner/validator per unit of gas. Denominated in wei',
COMMENT COLUMN basefeepergas 'The fee per unit of gas paid and burned for the curblocknumber. This fee is algorithmically determined. Denominated in wei',
COMMENT COLUMN dropreason 'If the transaction was dropped from the mempool, this describes the contextual reason for the drop',
COMMENT COLUMN rejectionreason 'If the transaction was rejected from the mempool, this describes the contextual reason for the rejection',
COMMENT COLUMN stuck 'A transaction was detected in the queued area of the mempool and is not eligible for inclusion in a block',
COMMENT COLUMN gasused 'If the transaction was published on-chain, this value indicates the amount of gas that was actually consumed. Denominated in wei';

CREATE TABLE block_native_mempool_transaction on cluster '{cluster}' AS block_native_mempool_transaction_local
ENGINE = Distributed('{cluster}', default, block_native_mempool_transaction_local, rand());
