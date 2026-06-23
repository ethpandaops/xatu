CREATE TABLE IF NOT EXISTS canonical_beacon_state_pending_partial_withdrawal_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number the pending partial withdrawal queue snapshot is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `state_id` LowCardinality(String) COMMENT 'The state ID the pending partial withdrawal was requested against',
    `position_in_queue` UInt32 COMMENT 'The index of the withdrawal within the pending partial withdrawals queue' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index the withdrawal applies to' CODEC(DoubleDelta, ZSTD(1)),
    `amount` UInt64 COMMENT 'The partial withdrawal amount in gwei' CODEC(ZSTD(1)),
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
