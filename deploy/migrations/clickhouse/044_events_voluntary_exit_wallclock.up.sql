CREATE TABLE tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'The index of the validator making the voluntary exit',
    `signature` String COMMENT 'The signature of the voluntary exit in the beacon API event stream payload',
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.',
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client that generated the event',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client that generated the event',
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client that generated the event',
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client that generated the event',
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client that generated the event',
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client that generated the event',
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client that generated the event',
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client that generated the event',
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event',
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event',
    `meta_network_id` Int32 COMMENT 'Ethereum network ID',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Ethereum consensus client version that generated the event',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Ethereum consensus client major version that generated the event',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Ethereum consensus client minor version that generated the event',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Ethereum consensus client patch version that generated the event',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Ethereum consensus client implementation that generated the event',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tmp/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(wallclock_epoch_start_date_time)
ORDER BY
    (
        wallclock_epoch_start_date_time,
        meta_network_name,
        meta_client_name,
        validator_index
    ) COMMENT 'Contains beacon API eventstream "voluntary exit" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_voluntary_exit_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_voluntary_exit_local,
    cityHash64(
        wallclock_epoch_start_date_time,
        meta_network_name,
        meta_client_name,
        validator_index
    )
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_voluntary_exit
WITH
    (CASE
        WHEN meta_network_name = 'mainnet' THEN 1606824023
        WHEN meta_network_name = 'sepolia' THEN 1655733600
        WHEN meta_network_name = 'holesky' THEN 1695902400
        WHEN meta_network_name = 'dencun-msf-1' THEN 1708607100
        ELSE 0
    END) AS genesis_time
SELECT
    NOW(),
    event_date_time,
    epoch,
    toDateTime(genesis_time + (epoch * 12 * 32)) AS epoch_start_date_time,
    floor((toUnixTimestamp(event_date_time) - genesis_time) / 12) AS wallclock_slot,
    toDateTime(genesis_time + (floor((toUnixTimestamp(event_date_time) - genesis_time) / 12) * 12)) AS wallclock_slot_start_date_time,
    floor((toUnixTimestamp(event_date_time) - genesis_time) / 12 / 32) AS wallclock_epoch,
    toDateTime(genesis_time + (floor((toUnixTimestamp(event_date_time) - genesis_time) / 12 / 32) * 12 * 32)) AS wallclock_epoch_start_date_time,
    validator_index,
    signature,
    meta_client_name,
    meta_client_id,
    meta_client_version,
    meta_client_implementation,
    meta_client_os,
    meta_client_ip,
    meta_client_geo_city,
    meta_client_geo_country,
    meta_client_geo_country_code,
    meta_client_geo_continent_code,
    meta_client_geo_longitude,
    meta_client_geo_latitude,
    meta_client_geo_autonomous_system_number,
    meta_client_geo_autonomous_system_organization,
    meta_network_id,
    meta_network_name,
    meta_consensus_version,
    meta_consensus_version_major,
    meta_consensus_version_minor,
    meta_consensus_version_patch,
    meta_consensus_implementation,
    meta_labels
FROM
    default.beacon_api_eth_v1_events_voluntary_exit_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_voluntary_exit_local
AND tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_voluntary_exit_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_voluntary_exit_local,
    cityHash64(
        wallclock_epoch_start_date_time,
        meta_network_name,
        meta_client_name,
        validator_index
    )
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' SYNC;
