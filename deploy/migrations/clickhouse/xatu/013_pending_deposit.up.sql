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
