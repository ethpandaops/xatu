DROP TABLE IF EXISTS default.canonical_beacon_validators on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_beacon_validators_local on cluster '{cluster}' SYNC;

CREATE TABLE default.canonical_beacon_validators_local
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `index` UInt32 COMMENT 'The index of the validator' CODEC(ZSTD(1)),
    `balance` UInt64 COMMENT 'The balance of the validator' CODEC(ZSTD(1)),
    `status` LowCardinality(String) COMMENT 'The status of the validator',
    `effective_balance` UInt64 COMMENT 'The effective balance of the validator' CODEC(ZSTD(1)),
    `slashed` Bool COMMENT 'Whether the validator is slashed',
    `activation_epoch` UInt64 COMMENT 'The epoch when the validator was activated' CODEC(ZSTD(1)),
    `activation_eligibility_epoch` UInt64 COMMENT 'The epoch when the validator was activated' CODEC(ZSTD(1)),
    `exit_epoch` UInt64 COMMENT 'The epoch when the validator exited' CODEC(ZSTD(1)),
    `withdrawable_epoch` UInt64 COMMENT 'The epoch when the validator can withdraw' CODEC(ZSTD(1)),
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
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Ethereum consensus client version that generated the event',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Ethereum consensus client major version that generated the event',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Ethereum consensus client minor version that generated the event',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Ethereum consensus client patch version that generated the event',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Ethereum consensus client implementation that generated the event',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY (epoch_start_date_time, `index`, meta_network_name)
COMMENT 'Contains a validator state for an epoch.'

CREATE TABLE default.canonical_beacon_validators ON CLUSTER '{cluster}' AS default.canonical_beacon_validators_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_validators_local,
    cityHash64(
        epoch_start_date_time,
        `index`,
        meta_network_name
    )
);
