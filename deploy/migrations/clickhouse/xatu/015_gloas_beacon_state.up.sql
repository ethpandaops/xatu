-- Gloas (EIP-7732) beacon state snapshots derived by cannon.

-- ptc_member
CREATE TABLE IF NOT EXISTS canonical_beacon_state_ptc_member_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot the payload timeliness committee serves' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the slot belongs to' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the committee was read from',
    `position` UInt32 COMMENT 'The index of the member within the committee, equal to its bit index in a payload attestation aggregation_bits' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator holding this position, which can repeat within a committee' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, slot, position)
COMMENT 'Contains the ordered payload timeliness committee of each Gloas slot, one row per committee position.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_ptc_member ON CLUSTER '{cluster}'
AS canonical_beacon_state_ptc_member_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_ptc_member_local, cityHash64(slot_start_date_time, meta_network_name, slot, position))
COMMENT 'Contains the ordered payload timeliness committee of each Gloas slot, one row per committee position.';

-- builder
CREATE TABLE IF NOT EXISTS canonical_beacon_state_builder_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the builder registry snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the registry was read from',
    `builder_index` UInt64 COMMENT 'The index of the builder in the registry' CODEC(DoubleDelta, ZSTD(1)),
    `pubkey` FixedString(98) COMMENT 'The public key of the builder' CODEC(ZSTD(1)),
    `version` UInt8 COMMENT 'The builder version byte' CODEC(ZSTD(1)),
    `execution_address` FixedString(42) COMMENT 'The execution address of the builder' CODEC(ZSTD(1)),
    `balance` UInt64 COMMENT 'The builder balance in gwei' CODEC(ZSTD(1)),
    `deposit_epoch` UInt64 COMMENT 'The epoch in which the builder was added to the registry' CODEC(ZSTD(1)),
    `withdrawable_epoch` UInt64 COMMENT 'The epoch from which the builder can be withdrawn, FAR_FUTURE_EPOCH while no exit was initiated' CODEC(ZSTD(1)),
    `status` LowCardinality(String) COMMENT 'The builder status (pending, active or exited) evaluated against the finalized checkpoint of the state',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, builder_index)
COMMENT 'Contains the Gloas builder registry snapshot for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_builder ON CLUSTER '{cluster}'
AS canonical_beacon_state_builder_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_builder_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, builder_index))
COMMENT 'Contains the Gloas builder registry snapshot for a canonical beacon state epoch.';

-- builder_pending_payment
CREATE TABLE IF NOT EXISTS canonical_beacon_state_builder_pending_payment_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the builder pending payments snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the payments were read from',
    `payment_index` UInt32 COMMENT 'The index into builder_pending_payments, below SLOTS_PER_EPOCH for the previous epoch and above for the current epoch' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot the payment belongs to' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `weight` UInt64 COMMENT 'The attestation weight accumulated for the payment in gwei' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'The fee recipient of the payment withdrawal' CODEC(ZSTD(1)),
    `amount` UInt64 COMMENT 'The payment amount in gwei' CODEC(ZSTD(1)),
    `builder_index` UInt64 COMMENT 'The index of the paying builder' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The validator index of the slot proposer' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, payment_index)
COMMENT 'Contains the non-empty Gloas builder pending payments of a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_builder_pending_payment ON CLUSTER '{cluster}'
AS canonical_beacon_state_builder_pending_payment_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_builder_pending_payment_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, payment_index))
COMMENT 'Contains the non-empty Gloas builder pending payments of a canonical beacon state epoch.';

-- builder_pending_withdrawal
CREATE TABLE IF NOT EXISTS canonical_beacon_state_builder_pending_withdrawal_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the builder pending withdrawal queue snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the withdrawal was read from',
    `position_in_queue` UInt32 COMMENT 'The index of the withdrawal within the builder pending withdrawals queue' CODEC(DoubleDelta, ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'The fee recipient of the withdrawal' CODEC(ZSTD(1)),
    `amount` UInt64 COMMENT 'The withdrawal amount in gwei' CODEC(ZSTD(1)),
    `builder_index` UInt64 COMMENT 'The index of the builder the withdrawal is charged to' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(epoch_start_date_time))
ORDER BY (meta_network_name, epoch_start_date_time, epoch, position_in_queue)
COMMENT 'Contains the Gloas builder pending withdrawal queue snapshot for a canonical beacon state epoch.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_builder_pending_withdrawal ON CLUSTER '{cluster}'
AS canonical_beacon_state_builder_pending_withdrawal_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_builder_pending_withdrawal_local, cityHash64(epoch_start_date_time, meta_network_name, epoch, position_in_queue))
COMMENT 'Contains the Gloas builder pending withdrawal queue snapshot for a canonical beacon state epoch.';

-- execution_payload_availability
CREATE TABLE IF NOT EXISTS canonical_beacon_state_execution_payload_availability_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot the availability bit is for' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the slot belongs to' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the bit was read from',
    `available` Bool COMMENT 'Whether the slot execution payload was revealed and applied to the state' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, slot)
COMMENT 'Contains the Gloas execution payload availability bit of each slot, as recorded by the canonical beacon state.';

CREATE TABLE IF NOT EXISTS canonical_beacon_state_execution_payload_availability ON CLUSTER '{cluster}'
AS canonical_beacon_state_execution_payload_availability_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_state_execution_payload_availability_local, cityHash64(slot_start_date_time, meta_network_name, slot))
COMMENT 'Contains the Gloas execution payload availability bit of each slot, as recorded by the canonical beacon state.';
