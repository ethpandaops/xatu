-- admin schema (database-agnostic).
-- Applied to the `admin` database via the migration matrix.
-- Target database supplied at apply time do not hardcode it here.

CREATE TABLE IF NOT EXISTS cryo_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime CODEC(DoubleDelta, ZSTD(1)),
    `dataset` LowCardinality(String),
    `mode` LowCardinality(String),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY meta_network_name
ORDER BY (dataset, mode, meta_network_name)
COMMENT 'Tracks cryo dataset processing state per block';

CREATE TABLE IF NOT EXISTS execution_block_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `processor` LowCardinality(String) COMMENT 'The type of processor that processed the block',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `complete` UInt8,
    `task_count` UInt32
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY meta_network_name
ORDER BY (block_number, processor, meta_network_name)
COMMENT 'Tracks execution block processing state';

CREATE TABLE IF NOT EXISTS cryo ON CLUSTER '{cluster}'
AS cryo_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'cryo_local', cityHash64(dataset, mode, meta_network_name))
COMMENT 'Tracks cryo dataset processing state per block';

CREATE TABLE IF NOT EXISTS execution_block ON CLUSTER '{cluster}'
AS execution_block_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'execution_block_local', cityHash64(block_number, processor, meta_network_name))
COMMENT 'Tracks execution block processing state';
