CREATE TABLE IF NOT EXISTS canonical_beacon_block_reward_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The validator index of the block proposer' CODEC(ZSTD(1)),
    `total` UInt64 COMMENT 'The total block reward in gwei' CODEC(ZSTD(1)),
    `attestations` UInt64 COMMENT 'The reward from including attestations in gwei' CODEC(ZSTD(1)),
    `sync_aggregate` UInt64 COMMENT 'The reward from including the sync aggregate in gwei' CODEC(ZSTD(1)),
    `proposer_slashings` UInt64 COMMENT 'The reward from including proposer slashings in gwei' CODEC(ZSTD(1)),
    `attester_slashings` UInt64 COMMENT 'The reward from including attester slashings in gwei' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, slot, block_root)
COMMENT 'Contains the proposer reward breakdown for a canonical beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_reward ON CLUSTER '{cluster}'
AS canonical_beacon_block_reward_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_reward_local, cityHash64(slot_start_date_time, meta_network_name, slot, block_root))
COMMENT 'Contains the proposer reward breakdown for a canonical beacon block.';
