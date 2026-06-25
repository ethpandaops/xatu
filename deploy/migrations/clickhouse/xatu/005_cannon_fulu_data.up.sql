-- Cannon: Electra/Fulu beacon data tables (execution requests, rewards, state metadata, Electra queues).

-- execution_request_deposit
CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_deposit_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `position_in_block` UInt32 COMMENT 'The index of the deposit within the block execution requests' CODEC(DoubleDelta, ZSTD(1)),
    `pubkey` String COMMENT 'The public key of the validator from the deposit request' CODEC(ZSTD(1)),
    `withdrawal_credentials` FixedString(66) COMMENT 'The withdrawal credentials from the deposit request' CODEC(ZSTD(1)),
    `amount` UInt128 COMMENT 'The deposit amount in gwei' CODEC(ZSTD(1)),
    `signature` String COMMENT 'The deposit signature' CODEC(ZSTD(1)),
    `index` UInt64 COMMENT 'The deposit index from the deposit request' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, block_root, position_in_block)
COMMENT 'Contains an EIP-6110 execution request deposit from a beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_deposit ON CLUSTER '{cluster}'
AS canonical_beacon_block_execution_request_deposit_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_execution_request_deposit_local, cityHash64(slot_start_date_time, meta_network_name, block_root, position_in_block))
COMMENT 'Contains an EIP-6110 execution request deposit from a beacon block.';

-- execution_request_withdrawal
CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_withdrawal_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `position_in_block` UInt32 COMMENT 'The index of the withdrawal within the block execution requests' CODEC(DoubleDelta, ZSTD(1)),
    `source_address` FixedString(42) COMMENT 'The source address that initiated the withdrawal request' CODEC(ZSTD(1)),
    `validator_pubkey` String COMMENT 'The public key of the validator the withdrawal targets' CODEC(ZSTD(1)),
    `amount` UInt128 COMMENT 'The withdrawal amount in gwei' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, block_root, position_in_block)
COMMENT 'Contains an EIP-7002 execution request withdrawal from a beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_withdrawal ON CLUSTER '{cluster}'
AS canonical_beacon_block_execution_request_withdrawal_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_execution_request_withdrawal_local, cityHash64(slot_start_date_time, meta_network_name, block_root, position_in_block))
COMMENT 'Contains an EIP-7002 execution request withdrawal from a beacon block.';

-- execution_request_consolidation
CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_consolidation_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `position_in_block` UInt32 COMMENT 'The index of the consolidation within the block execution requests' CODEC(DoubleDelta, ZSTD(1)),
    `source_address` FixedString(42) COMMENT 'The source address that initiated the consolidation request' CODEC(ZSTD(1)),
    `source_pubkey` String COMMENT 'The public key of the consolidation source validator' CODEC(ZSTD(1)),
    `target_pubkey` String COMMENT 'The public key of the consolidation target validator' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, block_root, position_in_block)
COMMENT 'Contains an EIP-7251 execution request consolidation from a beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_consolidation ON CLUSTER '{cluster}'
AS canonical_beacon_block_execution_request_consolidation_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_execution_request_consolidation_local, cityHash64(slot_start_date_time, meta_network_name, block_root, position_in_block))
COMMENT 'Contains an EIP-7251 execution request consolidation from a beacon block.';

-- block_reward
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

-- attestation_reward
CREATE TABLE IF NOT EXISTS canonical_beacon_attestation_reward_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index the reward applies to' CODEC(DoubleDelta, ZSTD(1)),
    `head` Int64 COMMENT 'The reward for correctly attesting to the head in gwei' CODEC(ZSTD(1)),
    `target` Int64 COMMENT 'The reward for correctly attesting to the target in gwei' CODEC(ZSTD(1)),
    `source` Int64 COMMENT 'The reward for correctly attesting to the source in gwei' CODEC(ZSTD(1)),
    `inclusion_delay` Nullable(UInt64) COMMENT 'The reward for inclusion delay in gwei' CODEC(ZSTD(1)),
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

-- sync_committee_reward
CREATE TABLE IF NOT EXISTS canonical_beacon_sync_committee_reward_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the reward is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index the reward applies to' CODEC(DoubleDelta, ZSTD(1)),
    `reward` Int64 COMMENT 'The sync committee reward in gwei (negative when a penalty)' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, slot, block_root, validator_index)
COMMENT 'Contains per-member sync committee rewards for a canonical beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_sync_committee_reward ON CLUSTER '{cluster}'
AS canonical_beacon_sync_committee_reward_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_sync_committee_reward_local, cityHash64(slot_start_date_time, meta_network_name, slot, block_root, validator_index))
COMMENT 'Contains per-member sync committee rewards for a canonical beacon block.';

-- randao
CREATE TABLE IF NOT EXISTS canonical_beacon_state_randao_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the RANDAO mix is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the RANDAO mix was requested against',
    `randao` FixedString(66) COMMENT 'The RANDAO mix for the epoch' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch)
COMMENT 'Contains the RANDAO mix for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_randao ON CLUSTER '{cluster}'
AS canonical_beacon_state_randao_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_randao_local, cityHash64(epoch_start_date_time, meta_network_name, epoch))
COMMENT 'Contains the RANDAO mix for a canonical beacon state epoch.';

-- finality_checkpoint
CREATE TABLE IF NOT EXISTS canonical_beacon_state_finality_checkpoint_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the finality checkpoints are for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the finality checkpoints were requested against',
    `previous_justified_epoch` UInt32 COMMENT 'The previous justified checkpoint epoch' CODEC(DoubleDelta, ZSTD(1)),
    `previous_justified_root` FixedString(66) COMMENT 'The previous justified checkpoint root' CODEC(ZSTD(1)),
    `current_justified_epoch` UInt32 COMMENT 'The current justified checkpoint epoch' CODEC(DoubleDelta, ZSTD(1)),
    `current_justified_root` FixedString(66) COMMENT 'The current justified checkpoint root' CODEC(ZSTD(1)),
    `finalized_epoch` UInt32 COMMENT 'The finalized checkpoint epoch' CODEC(DoubleDelta, ZSTD(1)),
    `finalized_root` FixedString(66) COMMENT 'The finalized checkpoint root' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch)
COMMENT 'Contains the finality checkpoints for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_finality_checkpoint ON CLUSTER '{cluster}'
AS canonical_beacon_state_finality_checkpoint_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_finality_checkpoint_local, cityHash64(epoch_start_date_time, meta_network_name, epoch))
COMMENT 'Contains the finality checkpoints for a canonical beacon state epoch.';

-- pending_deposit
CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_deposit_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the pending deposit queue snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the pending deposit was requested against',
    `position_in_queue` UInt32 COMMENT 'The index of the deposit within the pending deposits queue' CODEC(DoubleDelta, ZSTD(1)),
    `pubkey` String COMMENT 'The public key of the validator from the pending deposit' CODEC(ZSTD(1)),
    `withdrawal_credentials` FixedString(66) COMMENT 'The withdrawal credentials from the pending deposit' CODEC(ZSTD(1)),
    `amount` UInt128 COMMENT 'The deposit amount in gwei' CODEC(ZSTD(1)),
    `signature` String COMMENT 'The deposit signature' CODEC(ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot at which the deposit was queued' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, position_in_queue)
COMMENT 'Contains the Electra pending deposit queue snapshot for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_deposit ON CLUSTER '{cluster}'
AS canonical_beacon_state_pending_deposit_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_pending_deposit_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, position_in_queue))
COMMENT 'Contains the Electra pending deposit queue snapshot for a canonical beacon state epoch.';

-- pending_partial_withdrawal
CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_partial_withdrawal_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the pending partial withdrawal queue snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the pending partial withdrawal was requested against',
    `position_in_queue` UInt32 COMMENT 'The index of the withdrawal within the pending partial withdrawals queue' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index the withdrawal applies to' CODEC(DoubleDelta, ZSTD(1)),
    `amount` UInt128 COMMENT 'The partial withdrawal amount in gwei' CODEC(ZSTD(1)),
    `withdrawable_epoch` UInt64 COMMENT 'The epoch at which the withdrawal becomes withdrawable' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, position_in_queue)
COMMENT 'Contains the Electra pending partial withdrawal queue snapshot for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_partial_withdrawal ON CLUSTER '{cluster}'
AS canonical_beacon_state_pending_partial_withdrawal_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_pending_partial_withdrawal_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, position_in_queue))
COMMENT 'Contains the Electra pending partial withdrawal queue snapshot for a canonical beacon state epoch.';

-- pending_consolidation
CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_consolidation_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the pending consolidation queue snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the pending consolidation was requested against',
    `position_in_queue` UInt32 COMMENT 'The index of the consolidation within the pending consolidations queue' CODEC(DoubleDelta, ZSTD(1)),
    `source_index` UInt32 COMMENT 'The validator index of the consolidation source' CODEC(DoubleDelta, ZSTD(1)),
    `target_index` UInt32 COMMENT 'The validator index of the consolidation target' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, position_in_queue)
COMMENT 'Contains the Electra pending consolidation queue snapshot for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_consolidation ON CLUSTER '{cluster}'
AS canonical_beacon_state_pending_consolidation_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_pending_consolidation_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, position_in_queue))
COMMENT 'Contains the Electra pending consolidation queue snapshot for a canonical beacon state epoch.';
