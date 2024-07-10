DROP TABLE IF EXISTS canonical_beacon_validators on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_validators_local on cluster '{cluster}' SYNC;

DROP TABLE IF EXISTS canonical_beacon_validators_pubkeys on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_validators_pubkeys_local on cluster '{cluster}' SYNC;

DROP TABLE IF EXISTS canonical_beacon_validators_withdrawal_credentials on cluster '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_validators_withdrawal_credentials_local on cluster '{cluster}' SYNC;

CREATE TABLE default.canonical_beacon_validators_local on cluster '{cluster}'
(
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    epoch UInt32 CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    `index` UInt32 CODEC(ZSTD(1)),
    balance UInt64 CODEC(ZSTD(1)),
    `status` LowCardinality(String),
    pubkey String CODEC(ZSTD(1)),
    withdrawal_credentials String CODEC(ZSTD(1)),
    effective_balance UInt64 CODEC(ZSTD(1)),
    slashed Bool,
    activation_epoch UInt64 CODEC(ZSTD(1)),
    activation_eligibility_epoch UInt64 CODEC(ZSTD(1)),
    exit_epoch UInt64 CODEC(ZSTD(1)),
    withdrawable_epoch UInt64 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String) CODEC(ZSTD(1))
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY (epoch_start_date_time, index, meta_network_name);

ALTER TABLE default.canonical_beacon_validators_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a validator state for an epoch.',
COMMENT COLUMN updated_date_time 'When this row was last updated',
COMMENT COLUMN event_date_time 'When the client fetched the beacon block from a beacon node',
COMMENT COLUMN epoch 'The epoch number from beacon block payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN `index` 'The index of the validator',
COMMENT COLUMN `balance` 'The balance of the validator',
COMMENT COLUMN `status` 'The status of the validator',
COMMENT COLUMN pubkey 'The public key of the validator',
COMMENT COLUMN withdrawal_credentials 'The withdrawal credentials of the validator',
COMMENT COLUMN effective_balance 'The effective balance of the validator',
COMMENT COLUMN slashed 'Whether the validator is slashed',
COMMENT COLUMN activation_epoch 'The epoch when the validator was activated',
COMMENT COLUMN activation_eligibility_epoch 'The epoch when the validator was activated',
COMMENT COLUMN exit_epoch 'The epoch when the validator exited',
COMMENT COLUMN withdrawable_epoch 'The epoch when the validator can withdraw',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN meta_client_id 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.',
COMMENT COLUMN meta_client_version 'Version of the client that generated the event',
COMMENT COLUMN meta_client_implementation 'Implementation of the client that generated the event',
COMMENT COLUMN meta_client_os 'Operating system of the client that generated the event',
COMMENT COLUMN meta_client_ip 'IP address of the client that generated the event',
COMMENT COLUMN meta_client_geo_city 'City of the client that generated the event',
COMMENT COLUMN meta_client_geo_country 'Country of the client that generated the event',
COMMENT COLUMN meta_client_geo_country_code 'Country code of the client that generated the event',
COMMENT COLUMN meta_client_geo_continent_code 'Continent code of the client that generated the event',
COMMENT COLUMN meta_client_geo_longitude 'Longitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_latitude 'Latitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_number 'Autonomous system number of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_organization 'Autonomous system organization of the client that generated the event',
COMMENT COLUMN meta_network_id 'Ethereum network ID',
COMMENT COLUMN meta_network_name 'Ethereum network name',
COMMENT COLUMN meta_consensus_version 'Ethereum consensus client version that generated the event',
COMMENT COLUMN meta_consensus_version_major 'Ethereum consensus client major version that generated the event',
COMMENT COLUMN meta_consensus_version_minor 'Ethereum consensus client minor version that generated the event',
COMMENT COLUMN meta_consensus_version_patch 'Ethereum consensus client patch version that generated the event',
COMMENT COLUMN meta_consensus_implementation 'Ethereum consensus client implementation that generated the event',
COMMENT COLUMN meta_labels 'Labels associated with the event';

CREATE TABLE canonical_beacon_validators on cluster '{cluster}' AS canonical_beacon_validators_local
ENGINE = Distributed('{cluster}', default, canonical_beacon_validators_local, cityHash64(epoch_start_date_time, `index`, meta_network_name));
