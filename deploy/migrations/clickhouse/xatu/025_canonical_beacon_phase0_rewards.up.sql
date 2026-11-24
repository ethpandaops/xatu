-- phase0_attestation_reward
CREATE TABLE IF NOT EXISTS canonical_beacon_phase0_attestation_reward_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch whose attestation duties the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index the reward applies to' CODEC(DoubleDelta, ZSTD(1)),
    `head` Int64 COMMENT 'The net reward for attesting to the head in gwei (negative when a penalty)' CODEC(ZSTD(1)),
    `target` Int64 COMMENT 'The net reward for attesting to the target in gwei (negative when a penalty)' CODEC(ZSTD(1)),
    `source` Int64 COMMENT 'The net reward for attesting to the source in gwei (negative when a penalty)' CODEC(ZSTD(1)),
    `inclusion_delay` UInt64 COMMENT 'The attester inclusion delay reward in gwei, excluding the share paid to the including block proposer' CODEC(ZSTD(1)),
    `inactivity` Int64 COMMENT 'The inactivity leak penalty in gwei (zero or negative)' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, validator_index)
COMMENT 'Contains per-validator phase0 attestation reward components for a canonical epoch, computed from beacon states with the consensus spec get_attestation_deltas.';

CREATE TABLE IF NOT EXISTS canonical_beacon_phase0_attestation_reward ON CLUSTER '{cluster}'
AS canonical_beacon_phase0_attestation_reward_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_phase0_attestation_reward_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, validator_index))
COMMENT 'Contains per-validator phase0 attestation reward components for a canonical epoch, computed from beacon states with the consensus spec get_attestation_deltas.';

-- phase0_block_reward
CREATE TABLE IF NOT EXISTS canonical_beacon_phase0_block_reward_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number of the block' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number of the block' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The validator index of the block proposer' CODEC(ZSTD(1)),
    `total` UInt64 COMMENT 'The total proposer reward for the block in gwei' CODEC(ZSTD(1)),
    `attestations` UInt64 COMMENT 'The proposer inclusion reward for the attestations the block was first to include in gwei, paid at the following epoch transitions' CODEC(ZSTD(1)),
    `proposer_slashings` UInt64 COMMENT 'The whistleblower reward from including proposer slashings in gwei' CODEC(ZSTD(1)),
    `attester_slashings` UInt64 COMMENT 'The whistleblower reward from including attester slashings in gwei' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, slot, block_root)
COMMENT 'Contains the proposer reward breakdown for a canonical phase0 beacon block, computed from beacon states.';

CREATE TABLE IF NOT EXISTS canonical_beacon_phase0_block_reward ON CLUSTER '{cluster}'
AS canonical_beacon_phase0_block_reward_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_phase0_block_reward_local, cityHash64(slot_start_date_time, meta_network_name, slot, block_root))
COMMENT 'Contains the proposer reward breakdown for a canonical phase0 beacon block, computed from beacon states.';
