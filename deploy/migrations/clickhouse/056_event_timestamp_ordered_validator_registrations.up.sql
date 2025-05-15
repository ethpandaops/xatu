-- Create temporary table with new partitioning and ordering
CREATE TABLE IF NOT EXISTS tmp.mev_relay_validator_registration_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the registration was fetched' CODEC(DoubleDelta, ZSTD(1)),
    `timestamp` Int64 COMMENT 'The timestamp of the registration' CODEC(DoubleDelta, ZSTD(1)),
    `relay_name` String COMMENT 'The relay that the registration was fetched from' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The validator index of the validator registration' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'The gas limit of the validator registration' CODEC(DoubleDelta, ZSTD(1)),
    `fee_recipient` String COMMENT 'The fee recipient of the validator registration' CODEC(ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number derived from the validator registration `timestamp` field' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The slot start time derived from the validator registration `timestamp` field' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number derived from the validator registration `timestamp` field' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The epoch start time derived from the validator registration `timestamp` field' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'The wallclock slot when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'The start time for the slot when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'The wallclock epoch when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'The start time for the wallclock epoch when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' CODEC(ZSTD(1)),
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client that generated the event',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}-v2/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(event_date_time)
ORDER BY
    (
        event_date_time,
        meta_network_name,
        meta_client_name,
        relay_name,
        validator_index,
        timestamp
    ) COMMENT 'Contains MEV relay validator registrations data.';

-- Delete the old distributed table
DROP TABLE IF EXISTS default.mev_relay_validator_registration ON CLUSTER '{cluster}';

-- Copy data from old table to new table. This doesn't seem to work on all shards even with ON CLUSTER. Needs to be done manually.
INSERT INTO tmp.mev_relay_validator_registration_local SELECT * FROM default.mev_relay_validator_registration_local;

-- Rename old tables to temporary names
RENAME TABLE default.mev_relay_validator_registration_local TO tmp.mev_relay_validator_registration_local_old_v2 ON CLUSTER '{cluster}';

-- Rename tmp table to final name
RENAME TABLE tmp.mev_relay_validator_registration_local TO default.mev_relay_validator_registration_local ON CLUSTER '{cluster}';

-- Create new distributed table
CREATE TABLE default.mev_relay_validator_registration ON CLUSTER '{cluster}' AS default.mev_relay_validator_registration_local ENGINE = Distributed(
    '{cluster}',
    default,
    mev_relay_validator_registration_local,
    cityHash64(
        slot,
        meta_network_name
    )
);
