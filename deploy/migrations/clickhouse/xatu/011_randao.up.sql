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
