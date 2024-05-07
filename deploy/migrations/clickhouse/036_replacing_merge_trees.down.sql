-- beacon_api_eth_v1_beacon_committee
CREATE TABLE tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    committee_index LowCardinality(String),
    validators Array(UInt32) CODEC(ZSTD(1)),
    epoch UInt32 CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
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
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/beacon/states/{state_id}/committees data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API committee payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN committee_index 'The committee index in the beacon API committee payload',
COMMENT COLUMN validators 'The validator indices in the beacon API committee payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API committee payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
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

CREATE TABLE tmp.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_beacon_committee_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_beacon_committee_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_beacon_committee
SELECT
    toInt64(cityHash64(toString(slot) || committee_index || toString(validators) || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_beacon_committee_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_beacon_committee_local AND tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_beacon_committee_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_beacon_committee_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_blob_sidecar
CREATE TABLE tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    propagation_slot_start_diff UInt32 CODEC(ZSTD(1)),
    epoch UInt32 CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    block_root FixedString(66) CODEC(ZSTD(1)),
    blob_index UInt64 CODEC(ZSTD(1)),
    kzg_commitment FixedString(98) CODEC(ZSTD(1)),
    versioned_hash FixedString(66) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
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
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "blob_sidecar" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API event stream payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN block_root 'The beacon block root hash in the beacon API event stream payload',
COMMENT COLUMN blob_index 'The index of blob sidecar in the beacon API event stream payload',
COMMENT COLUMN kzg_commitment 'The KZG commitment in the beacon API event stream payload',
COMMENT COLUMN versioned_hash 'The versioned hash in the beacon API event stream payload',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_blob_sidecar_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_blob_sidecar_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_blob_sidecar
SELECT
    toInt64(cityHash64(toString(slot) || toString(blob_index) || block_root || kzg_commitment || versioned_hash || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_blob_sidecar_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_blob_sidecar_local AND tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_blob_sidecar_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_blob_sidecar_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_block
CREATE TABLE tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    propagation_slot_start_diff UInt32 CODEC(ZSTD(1)),
    block FixedString(66) CODEC(ZSTD(1)),
    epoch UInt32 CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    execution_optimistic Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
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
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "block" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API event stream payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time',
COMMENT COLUMN block 'The beacon block root hash in the beacon API event stream payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN execution_optimistic 'If the attached beacon node is running in execution optimistic mode',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_block_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_block_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_block
SELECT
    toInt64(cityHash64(toString(slot) || block || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_block_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_block_local AND tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_block_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_block_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_chain_reorg
CREATE TABLE tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3),
    slot UInt32,
    slot_start_date_time DateTime,
    propagation_slot_start_diff UInt32,
    depth UInt16,
    old_head_block FixedString(66),
    new_head_block FixedString(66),
    old_head_state FixedString(66),
    new_head_state FixedString(66),
    epoch UInt32,
    epoch_start_date_time DateTime,
    execution_optimistic Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String,
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6),
    meta_client_geo_city LowCardinality(String),
    meta_client_geo_country LowCardinality(String),
    meta_client_geo_country_code LowCardinality(String),
    meta_client_geo_continent_code LowCardinality(String),
    meta_client_geo_longitude Nullable(Float64),
    meta_client_geo_latitude Nullable(Float64),
    meta_client_geo_autonomous_system_number Nullable(UInt32),
    meta_client_geo_autonomous_system_organization Nullable(String),
    meta_network_id Int32,
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "chain reorg" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'The slot number of the chain reorg event in the beacon API event stream payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the reorg slot started',
COMMENT COLUMN propagation_slot_start_diff 'Difference in slots between when the reorg occurred and when the sentry received the event',
COMMENT COLUMN depth 'The depth of the chain reorg in the beacon API event stream payload',
COMMENT COLUMN old_head_block 'The old head block root hash in the beacon API event stream payload',
COMMENT COLUMN new_head_block 'The new head block root hash in the beacon API event stream payload',
COMMENT COLUMN old_head_state 'The old head state root hash in the beacon API event stream payload',
COMMENT COLUMN new_head_state 'The new head state root hash in the beacon API event stream payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN execution_optimistic 'Whether the execution of the epoch was optimistic',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_chain_reorg_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_chain_reorg_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_chain_reorg
SELECT
    toInt64(cityHash64(toString(slot) || toString(depth) || old_head_block || new_head_block || old_head_state || new_head_state || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_chain_reorg_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_chain_reorg_local AND tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_chain_reorg_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_chain_reorg_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_contribution_and_proof
CREATE TABLE tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3),
    aggregator_index UInt32,
    contribution_slot UInt32,
    contribution_slot_start_date_time DateTime,
    contribution_propagation_slot_start_diff UInt32,
    contribution_beacon_block_root FixedString(66),
    contribution_subcommittee_index LowCardinality(String),
    contribution_aggregation_bits String,
    contribution_signature String,
    contribution_epoch UInt32,
    contribution_epoch_start_date_time DateTime,
    selection_proof String,
    signature String,
    meta_client_name LowCardinality(String),
    meta_client_id String,
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6),
    meta_client_geo_city LowCardinality(String),
    meta_client_geo_country LowCardinality(String),
    meta_client_geo_country_code LowCardinality(String),
    meta_client_geo_continent_code LowCardinality(String),
    meta_client_geo_longitude Nullable(Float64),
    meta_client_geo_latitude Nullable(Float64),
    meta_client_geo_autonomous_system_number Nullable(UInt32),
    meta_client_geo_autonomous_system_organization Nullable(String),
    meta_network_id Int32,
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(contribution_slot_start_date_time)
ORDER BY (contribution_slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "contribution and proof" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN aggregator_index 'The validator index of the aggregator in the beacon API event stream payload',
COMMENT COLUMN contribution_slot 'The slot number of the contribution in the beacon API event stream payload',
COMMENT COLUMN contribution_slot_start_date_time 'The wall clock time when the contribution slot started',
COMMENT COLUMN contribution_propagation_slot_start_diff 'Difference in slots between when the contribution occurred and when the sentry received the event',
COMMENT COLUMN contribution_beacon_block_root 'The beacon block root hash in the beacon API event stream payload',
COMMENT COLUMN contribution_subcommittee_index 'The subcommittee index of the contribution in the beacon API event stream payload',
COMMENT COLUMN contribution_aggregation_bits 'The aggregation bits of the contribution in the beacon API event stream payload',
COMMENT COLUMN contribution_signature 'The signature of the contribution in the beacon API event stream payload',
COMMENT COLUMN contribution_epoch 'The epoch number of the contribution in the beacon API event stream payload',
COMMENT COLUMN contribution_epoch_start_date_time 'The wall clock time when the contribution epoch started',
COMMENT COLUMN selection_proof 'The selection proof in the beacon API event stream payload',
COMMENT COLUMN signature 'The signature in the beacon API event stream payload',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_contribution_and_proof_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_contribution_and_proof_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_contribution_and_proof
SELECT
    toInt64(cityHash64(toString(aggregator_index) || toString(contribution_slot) || contribution_beacon_block_root || contribution_subcommittee_index || contribution_aggregation_bits || contribution_signature || selection_proof || signature || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_contribution_and_proof_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_contribution_and_proof_local AND tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_contribution_and_proof_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_contribution_and_proof_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_finalized_checkpoint
CREATE TABLE tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    block FixedString(66) CODEC(ZSTD(1)),
    state FixedString(66) CODEC(ZSTD(1)),
    epoch UInt32 CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    execution_optimistic Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
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
ORDER BY (epoch_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "finalized checkpoint" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN block 'The finalized block root hash in the beacon API event stream payload',
COMMENT COLUMN state 'The finalized state root hash in the beacon API event stream payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN execution_optimistic 'Whether the execution of the epoch was optimistic',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_finalized_checkpoint_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_finalized_checkpoint_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_finalized_checkpoint
SELECT
    toInt64(cityHash64(toString(epoch) || block || state || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_finalized_checkpoint_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_finalized_checkpoint_local AND tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_finalized_checkpoint_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_finalized_checkpoint_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_head
CREATE TABLE tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3),
    slot UInt32,
    slot_start_date_time DateTime,
    propagation_slot_start_diff UInt32,
    block FixedString(66),
    epoch UInt32,
    epoch_start_date_time DateTime,
    epoch_transition Bool,
    execution_optimistic Bool,
    previous_duty_dependent_root FixedString(66),
    current_duty_dependent_root FixedString(66),
    meta_client_name LowCardinality(String),
    meta_client_id String,
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6),
    meta_client_geo_city LowCardinality(String),
    meta_client_geo_country LowCardinality(String),
    meta_client_geo_country_code LowCardinality(String),
    meta_client_geo_continent_code LowCardinality(String),
    meta_client_geo_longitude Nullable(Float64),
    meta_client_geo_latitude Nullable(Float64),
    meta_client_geo_autonomous_system_number Nullable(UInt32),
    meta_client_geo_autonomous_system_organization Nullable(String),
    meta_network_id Int32,
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "head" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API event stream payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time',
COMMENT COLUMN block 'The beacon block root hash in the beacon API event stream payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN epoch_transition 'If the event is an epoch transition',
COMMENT COLUMN execution_optimistic 'If the attached beacon node is running in execution optimistic mode',
COMMENT COLUMN previous_duty_dependent_root 'The previous duty dependent root in the beacon API event stream payload',
COMMENT COLUMN current_duty_dependent_root 'The current duty dependent root in the beacon API event stream payload',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_head_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_head_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_head
SELECT
    toInt64(cityHash64(toString(slot) || block || previous_duty_dependent_root || current_duty_dependent_root || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_head_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_head_local AND tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_head_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_head_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_voluntary_exit
CREATE TABLE tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3),
    epoch UInt32,
    epoch_start_date_time DateTime,
    validator_index UInt32,
    signature String,
    meta_client_name LowCardinality(String),
    meta_client_id String,
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6),
    meta_client_geo_city LowCardinality(String),
    meta_client_geo_country LowCardinality(String),
    meta_client_geo_country_code LowCardinality(String),
    meta_client_geo_continent_code LowCardinality(String),
    meta_client_geo_longitude Nullable(Float64),
    meta_client_geo_latitude Nullable(Float64),
    meta_client_geo_autonomous_system_number Nullable(UInt32),
    meta_client_geo_autonomous_system_organization Nullable(String),
    meta_network_id Int32,
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY (epoch_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "voluntary exit" data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN epoch 'The epoch number in the beacon API event stream payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN validator_index 'The index of the validator making the voluntary exit',
COMMENT COLUMN signature 'The signature of the voluntary exit in the beacon API event stream payload',
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

CREATE TABLE tmp.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_voluntary_exit_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_events_voluntary_exit_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_events_voluntary_exit
SELECT
    toInt64(cityHash64(toString(epoch) || validator_index || signature || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_events_voluntary_exit_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_voluntary_exit_local AND tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_voluntary_exit_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_voluntary_exit_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_validator_attestation_data
CREATE TABLE tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3),
    slot UInt32,
    slot_start_date_time DateTime,
    committee_index LowCardinality(String),
    beacon_block_root FixedString(66),
    epoch UInt32,
    epoch_start_date_time DateTime,
    source_epoch UInt32,
    source_epoch_start_date_time DateTime,
    source_root FixedString(66),
    target_epoch UInt32,
    target_epoch_start_date_time DateTime,
    target_root FixedString(66),
    request_date_time DateTime,
    request_duration UInt32,
    request_slot_start_diff UInt32,
    meta_client_name LowCardinality(String),
    meta_client_id String,
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6),
    meta_client_geo_city LowCardinality(String),
    meta_client_geo_country LowCardinality(String),
    meta_client_geo_country_code LowCardinality(String),
    meta_client_geo_continent_code LowCardinality(String),
    meta_client_geo_longitude Nullable(Float64),
    meta_client_geo_latitude Nullable(Float64),
    meta_client_geo_autonomous_system_number Nullable(UInt32),
    meta_client_geo_autonomous_system_organization Nullable(String),
    meta_network_id Int32,
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API validator attestation data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API validator attestation data payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN committee_index 'The committee index in the beacon API validator attestation data payload',
COMMENT COLUMN beacon_block_root 'The beacon block root hash in the beacon API validator attestation data payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API validator attestation data payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN source_epoch 'The source epoch number in the beacon API validator attestation data payload',
COMMENT COLUMN source_epoch_start_date_time 'The wall clock time when the source epoch started',
COMMENT COLUMN source_root 'The source beacon block root hash in the beacon API validator attestation data payload',
COMMENT COLUMN target_epoch 'The target epoch number in the beacon API validator attestation data payload',
COMMENT COLUMN target_epoch_start_date_time 'The wall clock time when the target epoch started',
COMMENT COLUMN target_root 'The target beacon block root hash in the beacon API validator attestation data payload',
COMMENT COLUMN request_date_time 'When the request was sent to the beacon node',
COMMENT COLUMN request_duration 'The request duration in milliseconds',
COMMENT COLUMN request_slot_start_diff 'The difference between the request_date_time and the slot_start_date_time',
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

CREATE TABLE tmp.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_validator_attestation_data_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v1_validator_attestation_data_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v1_validator_attestation_data
SELECT
    toInt64(cityHash64(toString(slot) || committee_index || beacon_block_root || toString(source_epoch) || source_root || toString(target_epoch) || target_root || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v1_validator_attestation_data_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_validator_attestation_data_local AND tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_validator_attestation_data_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_validator_attestation_data_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v2_beacon_block
CREATE TABLE tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3),
    slot UInt32,
    slot_start_date_time DateTime,
    epoch UInt32,
    epoch_start_date_time DateTime,
    block_root FixedString(66),
    block_version LowCardinality(String),
    block_total_bytes Nullable(UInt32),
    block_total_bytes_compressed Nullable(UInt32),
    parent_root FixedString(66),
    state_root FixedString(66),
    proposer_index UInt32,
    eth1_data_block_hash FixedString(66),
    eth1_data_deposit_root FixedString(66),
    execution_payload_block_hash FixedString(66),
    execution_payload_block_number UInt32,
    execution_payload_fee_recipient String,
    execution_payload_state_root FixedString(66),
    execution_payload_parent_hash FixedString(66),
    execution_payload_transactions_count Nullable(UInt32),
    execution_payload_transactions_total_bytes Nullable(UInt32),
    execution_payload_transactions_total_bytes_compressed Nullable(UInt32),
    meta_client_name LowCardinality(String),
    meta_client_id String,
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6),
    meta_client_geo_city LowCardinality(String),
    meta_client_geo_country LowCardinality(String),
    meta_client_geo_country_code LowCardinality(String),
    meta_client_geo_continent_code LowCardinality(String),
    meta_client_geo_longitude Nullable(Float64),
    meta_client_geo_latitude Nullable(Float64),
    meta_client_geo_autonomous_system_number Nullable(UInt32),
    meta_client_geo_autonomous_system_organization Nullable(String),
    meta_network_id Int32,
    meta_network_name LowCardinality(String),
    meta_execution_fork_id_hash LowCardinality(String),
    meta_execution_fork_id_next LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v2/beacon/blocks/{block_id} data from each sentry client attached to a beacon node.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'When the sentry fetched the beacon block from a beacon node',
COMMENT COLUMN slot 'The slot number from beacon block payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the reorg slot started',
COMMENT COLUMN epoch 'The epoch number from beacon block payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN block_root 'The root hash of the beacon block',
COMMENT COLUMN block_version 'The version of the beacon block',
COMMENT COLUMN block_total_bytes 'The total bytes of the beacon block payload',
COMMENT COLUMN block_total_bytes_compressed 'The total bytes of the beacon block payload when compressed using snappy',
COMMENT COLUMN parent_root 'The root hash of the parent beacon block',
COMMENT COLUMN state_root 'The root hash of the beacon state at this block',
COMMENT COLUMN proposer_index 'The index of the validator that proposed the beacon block',
COMMENT COLUMN eth1_data_block_hash 'The block hash of the associated execution block',
COMMENT COLUMN eth1_data_deposit_root 'The root of the deposit tree in the associated execution block',
COMMENT COLUMN execution_payload_block_hash 'The block hash of the execution payload',
COMMENT COLUMN execution_payload_block_number 'The block number of the execution payload',
COMMENT COLUMN execution_payload_fee_recipient 'The recipient of the fee for this execution payload',
COMMENT COLUMN execution_payload_state_root 'The state root of the execution payload',
COMMENT COLUMN execution_payload_parent_hash 'The parent hash of the execution payload',
COMMENT COLUMN execution_payload_transactions_count 'The transaction count of the execution payload',
COMMENT COLUMN execution_payload_transactions_total_bytes 'The transaction total bytes of the execution payload',
COMMENT COLUMN execution_payload_transactions_total_bytes_compressed 'The transaction total bytes of the execution payload when compressed using snappy',
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
COMMENT COLUMN meta_execution_fork_id_hash 'The hash of the fork ID of the current Ethereum network',
COMMENT COLUMN meta_execution_fork_id_next 'The fork ID of the next planned Ethereum network upgrade',
COMMENT COLUMN meta_consensus_version 'Ethereum consensus client version that generated the event',
COMMENT COLUMN meta_consensus_version_major 'Ethereum consensus client major version that generated the event',
COMMENT COLUMN meta_consensus_version_minor 'Ethereum consensus client minor version that generated the event',
COMMENT COLUMN meta_consensus_version_patch 'Ethereum consensus client patch version that generated the event',
COMMENT COLUMN meta_consensus_implementation 'Ethereum consensus client implementation that generated the event',
COMMENT COLUMN meta_labels 'Labels associated with the event';

CREATE TABLE tmp.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v2_beacon_block_local
ENGINE = Distributed('{cluster}', tmp, beacon_api_eth_v2_beacon_block_local, unique_key);

INSERT INTO tmp.beacon_api_eth_v2_beacon_block
SELECT
    toInt64(cityHash64(toString(slot)|| block_root || block_version || parent_root || state_root || toString(proposer_index) || eth1_data_block_hash || execution_payload_block_hash || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.beacon_api_eth_v2_beacon_block_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' SYNC;

ALTER TABLE tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN meta_execution_fork_id_hash,
    DROP COLUMN meta_execution_fork_id_next;

EXCHANGE TABLES default.beacon_api_eth_v2_beacon_block_local AND tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' AS default.beacon_api_eth_v2_beacon_block_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v2_beacon_block_local, unique_key);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}' SYNC;

-- mempool_transaction
CREATE TABLE tmp.mempool_transaction_local ON CLUSTER '{cluster}' (
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    hash FixedString(66) CODEC(ZSTD(1)),
    from FixedString(42) CODEC(ZSTD(1)),
    to Nullable(FixedString(42)) CODEC(ZSTD(1)),
    nonce UInt64 CODEC(ZSTD(1)),
    gas_price UInt128 CODEC(ZSTD(1)),
    gas UInt64 CODEC(ZSTD(1)),
    gas_tip_cap Nullable(UInt128),
    gas_fee_cap Nullable(UInt128),
    value UInt128 CODEC(ZSTD(1)),
    type Nullable(UInt8),
    size UInt32 CODEC(ZSTD(1)),
    call_data_size UInt32 CODEC(ZSTD(1)),
    blob_gas Nullable(UInt64),
    blob_gas_fee_cap Nullable(UInt128),
    blob_hashes Array(String),
    blob_sidecars_size Nullable(UInt32),
    blob_sidecars_empty_size Nullable(UInt32),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String),
    meta_execution_fork_id_hash LowCardinality(String),
    meta_execution_fork_id_next LowCardinality(String),
    meta_labels Map(String, String) CODEC(ZSTD(1))
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(event_date_time)
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE tmp.mempool_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Each row represents a transaction that was seen in the mempool by a sentry client. Sentries can report the same transaction multiple times if it has been long enough since the last report.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'The time when the sentry saw the transaction in the mempool',
COMMENT COLUMN hash 'The hash of the transaction',
COMMENT COLUMN from 'The address of the account that sent the transaction',
COMMENT COLUMN to 'The address of the account that is the transaction recipient',
COMMENT COLUMN nonce 'The nonce of the sender account at the time of the transaction',
COMMENT COLUMN gas_price 'The gas price of the transaction in wei',
COMMENT COLUMN gas 'The maximum gas provided for the transaction execution',
COMMENT COLUMN gas_tip_cap 'The priority fee (tip) the user has set for the transaction',
COMMENT COLUMN gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN value 'The value transferred with the transaction in wei',
COMMENT COLUMN type 'The type of the transaction',
COMMENT COLUMN size 'The size of the transaction data in bytes',
COMMENT COLUMN call_data_size 'The size of the call data of the transaction in bytes',
COMMENT COLUMN blob_gas 'The maximum gas provided for the blob transaction execution',
COMMENT COLUMN blob_gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_hashes 'The hashes of the blob commitments for blob transactions',
COMMENT COLUMN blob_sidecars_size 'The total size of the sidecars for blob transactions in bytes',
COMMENT COLUMN blob_sidecars_empty_size 'The total empty size of the sidecars for blob transactions in bytes',
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
COMMENT COLUMN meta_execution_fork_id_hash 'The hash of the fork ID of the current Ethereum network',
COMMENT COLUMN meta_execution_fork_id_next 'The fork ID of the next planned Ethereum network upgrade',
COMMENT COLUMN meta_labels 'Labels associated with the event';

CREATE TABLE tmp.mempool_transaction ON CLUSTER '{cluster}' AS tmp.mempool_transaction_local
ENGINE = Distributed('{cluster}', tmp, mempool_transaction_local, unique_key);

INSERT INTO tmp.mempool_transaction
SELECT
    toInt64(cityHash64(hash || from || to || toString(nonce) || toString(type) || meta_client_name) - 9223372036854775808),
    NOW(),
    *
FROM default.mempool_transaction_local;

DROP TABLE IF EXISTS default.mempool_transaction ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.mempool_transaction_local AND tmp.mempool_transaction_local ON CLUSTER '{cluster}';

CREATE TABLE default.mempool_transaction ON CLUSTER '{cluster}' AS default.mempool_transaction_local
ENGINE = Distributed('{cluster}', default, mempool_transaction_local, unique_key);

DROP TABLE IF EXISTS tmp.mempool_transaction ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS tmp.mempool_transaction_local ON CLUSTER '{cluster}' SYNC;
