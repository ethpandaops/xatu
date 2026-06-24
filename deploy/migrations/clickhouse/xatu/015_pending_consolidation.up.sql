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
