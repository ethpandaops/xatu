CREATE TABLE IF NOT EXISTS canonical_beacon_attestation_reward_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index the reward applies to' CODEC(DoubleDelta, ZSTD(1)),
    `head` Int64 COMMENT 'The reward for correctly attesting to the head in gwei' CODEC(ZSTD(1)),
    `target` Int64 COMMENT 'The reward for correctly attesting to the target in gwei' CODEC(ZSTD(1)),
    `source` Int64 COMMENT 'The reward for correctly attesting to the source in gwei' CODEC(ZSTD(1)),
    `inclusion_delay` UInt64 COMMENT 'The reward for inclusion delay in gwei' CODEC(ZSTD(1)),
    `inactivity` Int64 COMMENT 'The inactivity penalty in gwei' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, validator_index)
COMMENT 'Contains per-validator attestation reward components for a canonical epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_attestation_reward ON CLUSTER '{cluster}'
AS canonical_beacon_attestation_reward_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_attestation_reward_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, validator_index))
COMMENT 'Contains per-validator attestation reward components for a canonical epoch.';
