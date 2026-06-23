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
