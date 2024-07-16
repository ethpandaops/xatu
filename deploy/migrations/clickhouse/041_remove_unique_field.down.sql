-- beacon_api_eth_v1_beacon_committee
CREATE TABLE tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number in the beacon API committee payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `committee_index` LowCardinality(String) COMMENT 'The committee index in the beacon API committee payload',
    `validators` Array(UInt32) COMMENT 'The validator indices in the beacon API committee payload' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API committee payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API /eth/v1/beacon/states/{state_id}/committees data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_beacon_committee_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_beacon_committee_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_beacon_committee
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            committee_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    committee_index,
    validators,
    epoch,
    epoch_start_date_time,
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
    default.beacon_api_eth_v1_beacon_committee_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_beacon_committee_local
AND tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_beacon_committee_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_beacon_committee_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_blob_sidecar
CREATE TABLE tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number in the beacon API event stream payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'The difference between the event_date_time and the slot_start_date_time' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The beacon block root hash in the beacon API event stream payload' CODEC(ZSTD(1)),
    `blob_index` UInt64 COMMENT 'The index of blob sidecar in the beacon API event stream payload' CODEC(ZSTD(1)),
    `kzg_commitment` FixedString(98) COMMENT 'The KZG commitment in the beacon API event stream payload' CODEC(ZSTD(1)),
    `versioned_hash` FixedString(66) COMMENT 'The versioned hash in the beacon API event stream payload' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "blob_sidecar" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_blob_sidecar_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_blob_sidecar_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_blob_sidecar
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            block_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    propagation_slot_start_diff,
    epoch,
    epoch_start_date_time,
    block_root,
    blob_index,
    kzg_commitment,
    versioned_hash,
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
    default.beacon_api_eth_v1_events_blob_sidecar_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_blob_sidecar_local
AND tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_blob_sidecar_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_blob_sidecar_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_block
CREATE TABLE tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number in the beacon API event stream payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'The difference between the event_date_time and the slot_start_date_time' CODEC(ZSTD(1)),
    `block` FixedString(66) COMMENT 'The beacon block root hash in the beacon API event stream payload' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `execution_optimistic` Bool COMMENT 'If the attached beacon node is running in execution optimistic mode',
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "block" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_block_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_block_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_block
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            block
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    propagation_slot_start_diff,
    block,
    epoch,
    epoch_start_date_time,
    execution_optimistic,
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
    default.beacon_api_eth_v1_events_block_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_block_local
AND tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_block_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_block_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_chain_reorg
CREATE TABLE tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `slot` UInt32 COMMENT 'The slot number of the chain reorg event in the beacon API event stream payload',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the reorg slot started',
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference in slots between when the reorg occurred and when the sentry received the event',
    `depth` UInt16 COMMENT 'The depth of the chain reorg in the beacon API event stream payload',
    `old_head_block` FixedString(66) COMMENT 'The old head block root hash in the beacon API event stream payload',
    `new_head_block` FixedString(66) COMMENT 'The new head block root hash in the beacon API event stream payload',
    `old_head_state` FixedString(66) COMMENT 'The old head state root hash in the beacon API event stream payload',
    `new_head_state` FixedString(66) COMMENT 'The new head state root hash in the beacon API event stream payload',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
    `execution_optimistic` Bool COMMENT 'Whether the execution of the epoch was optimistic',
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "chain reorg" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_chain_reorg_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_chain_reorg_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_chain_reorg
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            old_head_block,
            new_head_block
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    propagation_slot_start_diff,
    depth,
    old_head_block,
    new_head_block,
    old_head_state,
    new_head_state,
    epoch,
    epoch_start_date_time,
    execution_optimistic,
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
    default.beacon_api_eth_v1_events_chain_reorg_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_chain_reorg_local
AND tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_chain_reorg_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_chain_reorg_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_contribution_and_proof
CREATE TABLE tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `aggregator_index` UInt32 COMMENT 'The validator index of the aggregator in the beacon API event stream payload',
    `contribution_slot` UInt32 COMMENT 'The slot number of the contribution in the beacon API event stream payload',
    `contribution_slot_start_date_time` DateTime COMMENT 'The wall clock time when the contribution slot started',
    `contribution_propagation_slot_start_diff` UInt32 COMMENT 'Difference in slots between when the contribution occurred and when the sentry received the event',
    `contribution_beacon_block_root` FixedString(66) COMMENT 'The beacon block root hash in the beacon API event stream payload',
    `contribution_subcommittee_index` LowCardinality(String) COMMENT 'The subcommittee index of the contribution in the beacon API event stream payload',
    `contribution_aggregation_bits` String COMMENT 'The aggregation bits of the contribution in the beacon API event stream payload',
    `contribution_signature` String COMMENT 'The signature of the contribution in the beacon API event stream payload',
    `contribution_epoch` UInt32 COMMENT 'The epoch number of the contribution in the beacon API event stream payload',
    `contribution_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the contribution epoch started',
    `selection_proof` String COMMENT 'The selection proof in the beacon API event stream payload',
    `signature` String COMMENT 'The signature in the beacon API event stream payload',
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(contribution_slot_start_date_time)
ORDER BY
    (
        contribution_slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "contribution and proof" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_contribution_and_proof_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_contribution_and_proof_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_contribution_and_proof
SELECT
    toInt64(
        cityHash64(
            contribution_slot_start_date_time,
            meta_network_name,
            meta_client_name,
            contribution_beacon_block_root,
            contribution_subcommittee_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    aggregator_index,
    contribution_slot,
    contribution_slot_start_date_time,
    contribution_propagation_slot_start_diff,
    contribution_beacon_block_root,
    contribution_subcommittee_index,
    contribution_aggregation_bits,
    contribution_signature,
    contribution_epoch,
    contribution_epoch_start_date_time,
    selection_proof,
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
    default.beacon_api_eth_v1_events_contribution_and_proof_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_contribution_and_proof_local
AND tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_contribution_and_proof_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_contribution_and_proof_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_finalized_checkpoint
CREATE TABLE tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `block` FixedString(66) COMMENT 'The finalized block root hash in the beacon API event stream payload',
    `state` FixedString(66) COMMENT 'The finalized state root hash in the beacon API event stream payload',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `execution_optimistic` Bool COMMENT 'Whether the execution of the epoch was optimistic',
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY
    (
        epoch_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "finalized checkpoint" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_finalized_checkpoint_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_finalized_checkpoint
SELECT
    toInt64(
        cityHash64(
            epoch_start_date_time,
            meta_network_name,
            meta_client_name,
            block,
            state
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    block,
    state,
    epoch,
    epoch_start_date_time,
    execution_optimistic,
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
    default.beacon_api_eth_v1_events_finalized_checkpoint_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_finalized_checkpoint_local
AND tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_finalized_checkpoint_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_finalized_checkpoint_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_head
CREATE TABLE tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `slot` UInt32 COMMENT 'Slot number in the beacon API event stream payload',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started',
    `propagation_slot_start_diff` UInt32 COMMENT 'The difference between the event_date_time and the slot_start_date_time',
    `block` FixedString(66) COMMENT 'The beacon block root hash in the beacon API event stream payload',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
    `epoch_transition` Bool COMMENT 'If the event is an epoch transition',
    `execution_optimistic` Bool COMMENT 'If the attached beacon node is running in execution optimistic mode',
    `previous_duty_dependent_root` FixedString(66) COMMENT 'The previous duty dependent root in the beacon API event stream payload',
    `current_duty_dependent_root` FixedString(66) COMMENT 'The current duty dependent root in the beacon API event stream payload',
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "head" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_head_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_head_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_head
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            block,
            previous_duty_dependent_root,
            current_duty_dependent_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    propagation_slot_start_diff,
    block,
    epoch,
    epoch_start_date_time,
    epoch_transition,
    execution_optimistic,
    previous_duty_dependent_root,
    current_duty_dependent_root,
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
    default.beacon_api_eth_v1_events_head_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_events_head_local
AND tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_head_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_head_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_events_voluntary_exit
CREATE TABLE tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY
    (
        epoch_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API eventstream "voluntary exit" data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_events_voluntary_exit_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_events_voluntary_exit_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_events_voluntary_exit
SELECT
    toInt64(
        cityHash64(
            epoch_start_date_time,
            meta_network_name,
            meta_client_name,
            validator_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    epoch,
    epoch_start_date_time,
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
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_proposer_duty
CREATE TABLE tmp.beacon_api_eth_v1_proposer_duty_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `proposer_validator_index` UInt32 COMMENT 'The validator index from the proposer duty payload' CODEC(ZSTD(1)),
    `proposer_pubkey` String COMMENT 'The BLS public key of the validator from the proposer duty payload' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains a proposer duty from a beacon block.';

CREATE TABLE tmp.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_proposer_duty_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_proposer_duty_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_proposer_duty
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            proposer_validator_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    proposer_validator_index,
    proposer_pubkey,
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
    default.beacon_api_eth_v1_proposer_duty_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_proposer_duty_local
AND tmp.beacon_api_eth_v1_proposer_duty_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_proposer_duty_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_proposer_duty_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_proposer_duty_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v1_validator_attestation_data
CREATE TABLE tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node',
    `slot` UInt32 COMMENT 'Slot number in the beacon API validator attestation data payload',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started',
    `committee_index` LowCardinality(String) COMMENT 'The committee index in the beacon API validator attestation data payload',
    `beacon_block_root` FixedString(66) COMMENT 'The beacon block root hash in the beacon API validator attestation data payload',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API validator attestation data payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
    `source_epoch` UInt32 COMMENT 'The source epoch number in the beacon API validator attestation data payload',
    `source_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the source epoch started',
    `source_root` FixedString(66) COMMENT 'The source beacon block root hash in the beacon API validator attestation data payload',
    `target_epoch` UInt32 COMMENT 'The target epoch number in the beacon API validator attestation data payload',
    `target_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the target epoch started',
    `target_root` FixedString(66) COMMENT 'The target beacon block root hash in the beacon API validator attestation data payload',
    `request_date_time` DateTime COMMENT 'When the request was sent to the beacon node',
    `request_duration` UInt32 COMMENT 'The request duration in milliseconds',
    `request_slot_start_diff` UInt32 COMMENT 'The difference between the request_date_time and the slot_start_date_time',
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API validator attestation data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v1_validator_attestation_data_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v1_validator_attestation_data_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v1_validator_attestation_data
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            committee_index,
            beacon_block_root,
            source_root,
            target_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    committee_index,
    beacon_block_root,
    epoch,
    epoch_start_date_time,
    source_epoch,
    source_epoch_start_date_time,
    source_root,
    target_epoch,
    target_epoch_start_date_time,
    target_root,
    request_date_time,
    request_duration,
    request_slot_start_diff,
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
    default.beacon_api_eth_v1_validator_attestation_data_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v1_validator_attestation_data_local
AND tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_validator_attestation_data_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_validator_attestation_data_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}' SYNC;

-- beacon_api_eth_v2_beacon_block
CREATE TABLE tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry fetched the beacon block from a beacon node',
    `slot` UInt32 COMMENT 'The slot number from beacon block payload',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the reorg slot started',
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block',
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `block_total_bytes` Nullable(UInt32) COMMENT 'The total bytes of the beacon block payload',
    `block_total_bytes_compressed` Nullable(UInt32) COMMENT 'The total bytes of the beacon block payload when compressed using snappy',
    `parent_root` FixedString(66) COMMENT 'The root hash of the parent beacon block',
    `state_root` FixedString(66) COMMENT 'The root hash of the beacon state at this block',
    `proposer_index` UInt32 COMMENT 'The index of the validator that proposed the beacon block',
    `eth1_data_block_hash` FixedString(66) COMMENT 'The block hash of the associated execution block',
    `eth1_data_deposit_root` FixedString(66) COMMENT 'The root of the deposit tree in the associated execution block',
    `execution_payload_block_hash` FixedString(66) COMMENT 'The block hash of the execution payload',
    `execution_payload_block_number` UInt32 COMMENT 'The block number of the execution payload',
    `execution_payload_fee_recipient` String COMMENT 'The recipient of the fee for this execution payload',
    `execution_payload_state_root` FixedString(66) COMMENT 'The state root of the execution payload',
    `execution_payload_parent_hash` FixedString(66) COMMENT 'The parent hash of the execution payload',
    `execution_payload_transactions_count` Nullable(UInt32) COMMENT 'The transaction count of the execution payload',
    `execution_payload_transactions_total_bytes` Nullable(UInt32) COMMENT 'The transaction total bytes of the execution payload',
    `execution_payload_transactions_total_bytes_compressed` Nullable(UInt32) COMMENT 'The transaction total bytes of the execution payload when compressed using snappy',
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon API /eth/v2/beacon/blocks/{block_id} data from each sentry client attached to a beacon node.';

CREATE TABLE tmp.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' AS tmp.beacon_api_eth_v2_beacon_block_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_api_eth_v2_beacon_block_local,
    unique_key
);

INSERT INTO
    tmp.beacon_api_eth_v2_beacon_block
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            block_root,
            parent_root,
            state_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    block_total_bytes,
    block_total_bytes_compressed,
    parent_root,
    state_root,
    proposer_index,
    eth1_data_block_hash,
    eth1_data_deposit_root,
    execution_payload_block_hash,
    execution_payload_block_number,
    execution_payload_fee_recipient,
    execution_payload_state_root,
    execution_payload_parent_hash,
    execution_payload_transactions_count,
    execution_payload_transactions_total_bytes,
    execution_payload_transactions_total_bytes_compressed,
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
    default.beacon_api_eth_v2_beacon_block_local;

DROP TABLE IF EXISTS default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_api_eth_v2_beacon_block_local
AND tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' AS default.beacon_api_eth_v2_beacon_block_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v2_beacon_block_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}' SYNC;

-- beacon_block_classification
CREATE TABLE tmp.beacon_block_classification_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block classification',
    `slot` UInt32 COMMENT 'The slot number from beacon block classification',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block classification' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `best_guess_single` LowCardinality(String) COMMENT 'The best guess of the client that generated the beacon block',
    `best_guess_multi` LowCardinality(String) COMMENT 'The best guess of the clients that generated the beacon block. This value will typically equal the best_guess_single value, but when multiple clients have high probabilities, this value will have multiple eg. "prysm or lighthouse"',
    `client_probability_uncertain` Float32 COMMENT 'The probability that the client that generated the beacon block is uncertain' CODEC(ZSTD(1)),
    `client_probability_prysm` Float32 COMMENT 'The probability that the client that generated the beacon block is Prysm' CODEC(ZSTD(1)),
    `client_probability_teku` Float32 COMMENT 'The probability that the client that generated the beacon block is Teku' CODEC(ZSTD(1)),
    `client_probability_nimbus` Float32 COMMENT 'The probability that the client that generated the beacon block is Nimbus' CODEC(ZSTD(1)),
    `client_probability_lodestar` Float32 COMMENT 'The probability that the client that generated the beacon block is Lodestar' CODEC(ZSTD(1)),
    `client_probability_grandine` Float32 COMMENT 'The probability that the client that generated the beacon block is Grandine' CODEC(ZSTD(1)),
    `client_probability_lighthouse` Float32 COMMENT 'The probability that the client that generated the beacon block is Lighthouse' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The index of the validator that proposed the beacon block' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon block classification for a given slot. This is a best guess based on the client probabilities of the proposer. This is not guaranteed to be correct.';

CREATE TABLE tmp.beacon_block_classification ON CLUSTER '{cluster}' AS tmp.beacon_block_classification_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_block_classification_local,
    unique_key
);

INSERT INTO
    tmp.beacon_block_classification
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            proposer_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    best_guess_single,
    best_guess_multi,
    client_probability_uncertain,
    client_probability_prysm,
    client_probability_teku,
    client_probability_nimbus,
    client_probability_lodestar,
    client_probability_grandine,
    client_probability_lighthouse,
    proposer_index,
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
    default.beacon_block_classification_local;

DROP TABLE IF EXISTS default.beacon_block_classification ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_block_classification_local
AND tmp.beacon_block_classification_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_block_classification ON CLUSTER '{cluster}' AS default.beacon_block_classification_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_block_classification_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_block_classification ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_block_classification_local ON CLUSTER '{cluster}' SYNC;

-- beacon_p2p_attestation
CREATE TABLE tmp.beacon_p2p_attestation_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node',
    `slot` UInt32 COMMENT 'Slot number in the beacon P2P payload',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started',
    `propagation_slot_start_diff` UInt32 COMMENT 'The difference between the event_date_time and the slot_start_date_time' CODEC(ZSTD(1)),
    `committee_index` LowCardinality(String) COMMENT 'The committee index in the beacon P2P payload',
    `attesting_validator_index` Nullable(UInt32) COMMENT 'The index of the validator attesting to the event' CODEC(ZSTD(1)),
    `attesting_validator_committee_index` LowCardinality(String) COMMENT 'The committee index of the attesting validator',
    `aggregation_bits` String COMMENT 'The aggregation bits of the event in the beacon P2P payload' CODEC(ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'The beacon block root hash in the beacon P2P payload' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number in the beacon P2P payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `source_epoch` UInt32 COMMENT 'The source epoch number in the beacon P2P payload' CODEC(DoubleDelta, ZSTD(1)),
    `source_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the source epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `source_root` FixedString(66) COMMENT 'The source beacon block root hash in the beacon P2P payload' CODEC(ZSTD(1)),
    `target_epoch` UInt32 COMMENT 'The target epoch number in the beacon P2P payload' CODEC(DoubleDelta, ZSTD(1)),
    `target_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the target epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `target_root` FixedString(66) COMMENT 'The target beacon block root hash in the beacon P2P payload' CODEC(ZSTD(1)),
    `attestation_subnet` LowCardinality(String) COMMENT 'The attestation subnet the attestation was gossiped on',
    `validated` Bool COMMENT 'Whether the attestation was validated by the client',
    `peer_id` String COMMENT 'The originating peer ID for the gossiped data' CODEC(ZSTD(1)),
    `peer_latency` UInt32 COMMENT 'The latency of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_version` LowCardinality(String) COMMENT 'Peer client version that gossiped the data',
    `peer_version_major` LowCardinality(String) COMMENT 'Peer client major version that gossiped the data',
    `peer_version_minor` LowCardinality(String) COMMENT 'Peer client minor version that gossiped the data',
    `peer_version_patch` LowCardinality(String) COMMENT 'Peer client patch version that gossiped the data',
    `peer_implementation` LowCardinality(String) COMMENT 'Peer client implementation that gossiped the data',
    `peer_ip` Nullable(IPv6) COMMENT 'IP address of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_city` LowCardinality(String) COMMENT 'City of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_country` LowCardinality(String) COMMENT 'Country of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_country_code` LowCardinality(String) COMMENT 'Country code of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the peer that gossiped the data' CODEC(ZSTD(1)),
    `peer_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the peer that gossiped the data' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains beacon chain P2P "attestation" data';

CREATE TABLE tmp.beacon_p2p_attestation ON CLUSTER '{cluster}' AS tmp.beacon_p2p_attestation_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    beacon_p2p_attestation_local,
    unique_key
);

INSERT INTO
    tmp.beacon_p2p_attestation
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            committee_index,
            beacon_block_root,
            source_root,
            target_root,
            attestation_subnet,
            peer_id
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    propagation_slot_start_diff,
    committee_index,
    attesting_validator_index,
    attesting_validator_committee_index,
    aggregation_bits,
    beacon_block_root,
    epoch,
    epoch_start_date_time,
    source_epoch,
    source_epoch_start_date_time,
    source_root,
    target_epoch,
    target_epoch_start_date_time,
    target_root,
    attestation_subnet,
    validated,
    peer_id,
    peer_latency,
    peer_version,
    peer_version_major,
    peer_version_minor,
    peer_version_patch,
    peer_implementation,
    peer_ip,
    peer_geo_city,
    peer_geo_country,
    peer_geo_country_code,
    peer_geo_continent_code,
    peer_geo_longitude,
    peer_geo_latitude,
    peer_geo_autonomous_system_number,
    peer_geo_autonomous_system_organization,
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
    default.beacon_p2p_attestation_local;

DROP TABLE IF EXISTS default.beacon_p2p_attestation ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.beacon_p2p_attestation_local
AND tmp.beacon_p2p_attestation_local ON CLUSTER '{cluster}';

CREATE TABLE default.beacon_p2p_attestation ON CLUSTER '{cluster}' AS default.beacon_p2p_attestation_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_p2p_attestation_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.beacon_p2p_attestation ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.beacon_p2p_attestation_local ON CLUSTER '{cluster}' SYNC;

-- block_native_mempool_transaction
CREATE TABLE tmp.block_native_mempool_transaction_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `detecttime` DateTime64(3) COMMENT 'Timestamp that the transaction was detected in mempool' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'Unique identifier hash for a given transaction' CODEC(ZSTD(1)),
    `status` LowCardinality(String) COMMENT 'Status of the transaction',
    `region` LowCardinality(String) COMMENT 'The geographic region for the node that detected the transaction',
    `reorg` Nullable(FixedString(66)) COMMENT 'If there was a reorg, refers to the blockhash of the reorg' CODEC(ZSTD(1)),
    `replace` Nullable(FixedString(66)) COMMENT 'If the transaction was replaced (speedup/cancel), the transaction hash of the replacement' CODEC(ZSTD(1)),
    `curblocknumber` Nullable(UInt64) COMMENT 'The block number the event was detected in' CODEC(ZSTD(1)),
    `failurereason` Nullable(String) COMMENT 'If a transaction failed, this field provides contextual information' CODEC(ZSTD(1)),
    `blockspending` Nullable(UInt64) COMMENT 'If a transaction was finalized (confirmed, failed), this refers to the number of blocks that the transaction was waiting to get on-chain' CODEC(ZSTD(1)),
    `timepending` Nullable(UInt64) COMMENT 'If a transaction was finalized (confirmed, failed), this refers to the time in milliseconds that the transaction was waiting to get on-chain' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'A unique number which counts the number of transactions sent from a given address' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'The maximum number of gas units allowed for the transaction' CODEC(ZSTD(1)),
    `gasprice` UInt128 COMMENT 'The price offered to the miner/validator per unit of gas. Denominated in wei' CODEC(ZSTD(1)),
    `value` UInt128 COMMENT 'The amount of ETH transferred or sent to contract. Denominated in wei' CODEC(ZSTD(1)),
    `toaddress` Nullable(FixedString(42)) COMMENT 'The destination of a given transaction' CODEC(ZSTD(1)),
    `fromaddress` FixedString(42) COMMENT 'The source/initiator of a given transaction' CODEC(ZSTD(1)),
    `datasize` UInt32 COMMENT 'The size of the call data of the transaction in bytes' CODEC(ZSTD(1)),
    `data4bytes` Nullable(FixedString(10)) COMMENT 'The first 4 bytes of the call data of the transaction' CODEC(ZSTD(1)),
    `network` LowCardinality(String) COMMENT 'The specific Ethereum network used',
    `type` UInt8 COMMENT '"Post EIP-1559, this indicates how the gas parameters are submitted to the network: - type 0 - legacy - type 1 - usage of access lists according to EIP-2930 - type 2 - using maxpriorityfeepergas and maxfeepergas"' CODEC(ZSTD(1)),
    `maxpriorityfeepergas` Nullable(UInt128) COMMENT 'The maximum value for a tip offered to the miner/validator per unit of gas. The actual tip paid can be lower if (maxfee - basefee) < maxpriorityfee. Denominated in wei' CODEC(ZSTD(1)),
    `maxfeepergas` Nullable(UInt128) COMMENT 'The maximum value for the transaction fee (including basefee and tip) offered to the miner/validator per unit of gas. Denominated in wei' CODEC(ZSTD(1)),
    `basefeepergas` Nullable(UInt128) COMMENT 'The fee per unit of gas paid and burned for the curblocknumber. This fee is algorithmically determined. Denominated in wei' CODEC(ZSTD(1)),
    `dropreason` Nullable(String) COMMENT 'If the transaction was dropped from the mempool, this describes the contextual reason for the drop' CODEC(ZSTD(1)),
    `rejectionreason` Nullable(String) COMMENT 'If the transaction was rejected from the mempool, this describes the contextual reason for the rejection' CODEC(ZSTD(1)),
    `stuck` Bool COMMENT 'A transaction was detected in the queued area of the mempool and is not eligible for inclusion in a block' CODEC(ZSTD(1)),
    `gasused` Nullable(UInt64) COMMENT 'If the transaction was published on-chain, this value indicates the amount of gas that was actually consumed. Denominated in wei' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(detecttime)
ORDER BY
    (
        detecttime,
        unique_key,
        network
    ) COMMENT 'Contains transactions from block native mempool dataset';

CREATE TABLE tmp.block_native_mempool_transaction ON CLUSTER '{cluster}' AS tmp.block_native_mempool_transaction_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    block_native_mempool_transaction_local,
    unique_key
);

INSERT INTO
    tmp.block_native_mempool_transaction
SELECT
    toInt64(
        cityHash64(
            detecttime,
            network,
            hash,
            fromaddress,
            nonce,
            gas
        ) - 9223372036854775808
    ),
    NOW(),
    detecttime,
    hash,
    status,
    region,
    reorg,
    replace,
    curblocknumber,
    failurereason,
    blockspending,
    timepending,
    nonce,
    gas,
    gasprice,
    value,
    toaddress,
    fromaddress,
    datasize,
    data4bytes,
    network,
    type,
    maxpriorityfeepergas,
    maxfeepergas,
    basefeepergas,
    dropreason,
    rejectionreason,
    stuck,
    gasused
FROM
    default.block_native_mempool_transaction_local;

DROP TABLE IF EXISTS default.block_native_mempool_transaction ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.block_native_mempool_transaction_local
AND tmp.block_native_mempool_transaction_local ON CLUSTER '{cluster}';

CREATE TABLE default.block_native_mempool_transaction ON CLUSTER '{cluster}' AS default.block_native_mempool_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    block_native_mempool_transaction_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.block_native_mempool_transaction ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.block_native_mempool_transaction_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_blob_sidecar
CREATE TABLE tmp.canonical_beacon_blob_sidecar_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_parent_root` FixedString(66) COMMENT 'The root hash of the parent beacon block' CODEC(ZSTD(1)),
    `versioned_hash` FixedString(66) COMMENT 'The versioned hash in the beacon API event stream payload' CODEC(ZSTD(1)),
    `kzg_commitment` FixedString(98) COMMENT 'The KZG commitment in the blob sidecar payload' CODEC(ZSTD(1)),
    `kzg_proof` FixedString(98) COMMENT 'The KZG proof in the blob sidecar payload' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The index of the validator that proposed the beacon block' CODEC(ZSTD(1)),
    `blob_index` UInt64 COMMENT 'The index of blob sidecar in the blob sidecar payload' CODEC(ZSTD(1)),
    `blob_size` UInt32 COMMENT 'The total bytes of the blob' CODEC(ZSTD(1)),
    `blob_empty_size` Nullable(UInt32) COMMENT 'The total empty size of the blob in bytes' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains a blob sidecar from a beacon block.';

CREATE TABLE tmp.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}' AS tmp.canonical_beacon_blob_sidecar_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_blob_sidecar_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_blob_sidecar
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            blob_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_parent_root,
    versioned_hash,
    kzg_commitment,
    kzg_proof,
    proposer_index,
    blob_index,
    blob_size,
    blob_empty_size,
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
    default.canonical_beacon_blob_sidecar_local;

DROP TABLE IF EXISTS default.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_blob_sidecar_local
AND tmp.canonical_beacon_blob_sidecar_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}' AS default.canonical_beacon_blob_sidecar_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_blob_sidecar_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_attester_slashing
CREATE TABLE tmp.canonical_beacon_block_attester_slashing_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `attestation_1_attesting_indices` Array(UInt32) COMMENT 'The attesting indices from the first attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_1_signature` String COMMENT 'The signature from the first attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_1_data_beacon_block_root` FixedString(66) COMMENT 'The beacon block root from the first attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_1_data_slot` UInt32 COMMENT 'The slot number from the first attestation in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `attestation_1_data_index` UInt32 COMMENT 'The attestor index from the first attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_1_data_source_epoch` UInt32 COMMENT 'The source epoch number from the first attestation in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `attestation_1_data_source_root` FixedString(66) COMMENT 'The source root from the first attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_1_data_target_epoch` UInt32 COMMENT 'The target epoch number from the first attestation in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `attestation_1_data_target_root` FixedString(66) COMMENT 'The target root from the first attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_2_attesting_indices` Array(UInt32) COMMENT 'The attesting indices from the second attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_2_signature` String COMMENT 'The signature from the second attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_2_data_beacon_block_root` FixedString(66) COMMENT 'The beacon block root from the second attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_2_data_slot` UInt32 COMMENT 'The slot number from the second attestation in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `attestation_2_data_index` UInt32 COMMENT 'The attestor index from the second attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_2_data_source_epoch` UInt32 COMMENT 'The source epoch number from the second attestation in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `attestation_2_data_source_root` FixedString(66) COMMENT 'The source root from the second attestation in the slashing payload' CODEC(ZSTD(1)),
    `attestation_2_data_target_epoch` UInt32 COMMENT 'The target epoch number from the second attestation in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `attestation_2_data_target_root` FixedString(66) COMMENT 'The target root from the second attestation in the slashing payload' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains attester slashing from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_attester_slashing_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_attester_slashing_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_attester_slashing
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            attestation_1_attesting_indices,
            attestation_2_attesting_indices,
            attestation_1_data_slot,
            attestation_2_data_slot,
            attestation_1_data_beacon_block_root,
            attestation_2_data_beacon_block_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    attestation_1_attesting_indices,
    attestation_1_signature,
    attestation_1_data_beacon_block_root,
    attestation_1_data_slot,
    attestation_1_data_index,
    attestation_1_data_source_epoch,
    attestation_1_data_source_root,
    attestation_1_data_target_epoch,
    attestation_1_data_target_root,
    attestation_2_attesting_indices,
    attestation_2_signature,
    attestation_2_data_beacon_block_root,
    attestation_2_data_slot,
    attestation_2_data_index,
    attestation_2_data_source_epoch,
    attestation_2_data_source_root,
    attestation_2_data_target_epoch,
    attestation_2_data_target_root,
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
    default.canonical_beacon_block_attester_slashing_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_attester_slashing_local
AND tmp.canonical_beacon_block_attester_slashing_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}' AS default.canonical_beacon_block_attester_slashing_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_attester_slashing_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_attester_slashing_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_bls_to_execution_change
CREATE TABLE tmp.canonical_beacon_block_bls_to_execution_change_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `exchanging_message_validator_index` UInt32 COMMENT 'The validator index from the exchanging message' CODEC(ZSTD(1)),
    `exchanging_message_from_bls_pubkey` String COMMENT 'The BLS public key from the exchanging message' CODEC(ZSTD(1)),
    `exchanging_message_to_execution_address` FixedString(42) COMMENT 'The execution address from the exchanging message' CODEC(ZSTD(1)),
    `exchanging_signature` String COMMENT 'The signature for the exchanging message' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains bls to execution change from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_bls_to_execution_change ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_bls_to_execution_change_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_bls_to_execution_change_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_bls_to_execution_change
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            exchanging_message_validator_index,
            exchanging_message_from_bls_pubkey,
            exchanging_message_to_execution_address
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    exchanging_message_validator_index,
    exchanging_message_from_bls_pubkey,
    exchanging_message_to_execution_address,
    exchanging_signature,
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
    default.canonical_beacon_block_bls_to_execution_change_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_bls_to_execution_change ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_bls_to_execution_change_local
AND tmp.canonical_beacon_block_bls_to_execution_change_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_bls_to_execution_change ON CLUSTER '{cluster}' AS default.canonical_beacon_block_bls_to_execution_change_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_bls_to_execution_change_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_bls_to_execution_change ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_bls_to_execution_change_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_deposit
CREATE TABLE tmp.canonical_beacon_block_deposit_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `deposit_proof` Array(String) COMMENT 'The proof of the deposit data' CODEC(ZSTD(1)),
    `deposit_data_pubkey` String COMMENT 'The BLS public key of the validator from the deposit data' CODEC(ZSTD(1)),
    `deposit_data_withdrawal_credentials` FixedString(66) COMMENT 'The withdrawal credentials of the validator from the deposit data' CODEC(ZSTD(1)),
    `deposit_data_amount` UInt128 COMMENT 'The amount of the deposit from the deposit data' CODEC(ZSTD(1)),
    `deposit_data_signature` String COMMENT 'The signature of the deposit data' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains a deposit from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_deposit ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_deposit_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_deposit_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_deposit
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            deposit_data_pubkey,
            deposit_proof
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    deposit_proof,
    deposit_data_pubkey,
    deposit_data_withdrawal_credentials,
    deposit_data_amount,
    deposit_data_signature,
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
    default.canonical_beacon_block_deposit_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_deposit ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_deposit_local
AND tmp.canonical_beacon_block_deposit_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_deposit ON CLUSTER '{cluster}' AS default.canonical_beacon_block_deposit_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_deposit_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_deposit ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_deposit_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_execution_transaction
CREATE TABLE tmp.canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `position` UInt32 COMMENT 'The position of the transaction in the beacon block' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'The hash of the transaction' CODEC(ZSTD(1)),
    `from` FixedString(42) COMMENT 'The address of the account that sent the transaction' CODEC(ZSTD(1)),
    `to` Nullable(FixedString(42)) COMMENT 'The address of the account that is the transaction recipient' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The nonce of the sender account at the time of the transaction' CODEC(ZSTD(1)),
    `gas_price` UInt128 COMMENT 'The gas price of the transaction in wei' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'The maximum gas provided for the transaction execution' CODEC(ZSTD(1)),
    `gas_tip_cap` Nullable(UInt128) COMMENT 'The priority fee (tip) the user has set for the transaction' CODEC(ZSTD(1)),
    `gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee the user has set for the transaction' CODEC(ZSTD(1)),
    `value` UInt128 COMMENT 'The value transferred with the transaction in wei' CODEC(ZSTD(1)),
    `type` UInt8 COMMENT 'The type of the transaction' CODEC(ZSTD(1)),
    `size` UInt32 COMMENT 'The size of the transaction data in bytes' CODEC(ZSTD(1)),
    `call_data_size` UInt32 COMMENT 'The size of the call data of the transaction in bytes' CODEC(ZSTD(1)),
    `blob_gas` Nullable(UInt64) COMMENT 'The maximum gas provided for the blob transaction execution' CODEC(ZSTD(1)),
    `blob_gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee the user has set for the transaction' CODEC(ZSTD(1)),
    `blob_hashes` Array(String) COMMENT 'The hashes of the blob commitments for blob transactions' CODEC(ZSTD(1)),
    `blob_sidecars_size` Nullable(UInt32) COMMENT 'The total size of the sidecars for blob transactions in bytes' CODEC(ZSTD(1)),
    `blob_sidecars_empty_size` Nullable(UInt32) COMMENT 'The total empty size of the sidecars for blob transactions in bytes' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains execution transaction from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_execution_transaction_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_execution_transaction_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_execution_transaction
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            position,
            hash,
            nonce
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    position,
    hash,
from
,
    to,
    nonce,
    gas_price,
    gas,
    gas_tip_cap,
    gas_fee_cap,
    value,
    type,
    size,
    call_data_size,
    blob_gas,
    blob_gas_fee_cap,
    blob_hashes,
    blob_sidecars_size,
    blob_sidecars_empty_size,
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
    default.canonical_beacon_block_execution_transaction_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_execution_transaction_local
AND tmp.canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}' AS default.canonical_beacon_block_execution_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_execution_transaction_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block
CREATE TABLE tmp.canonical_beacon_block_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `block_total_bytes` Nullable(UInt32) COMMENT 'The total bytes of the beacon block payload' CODEC(ZSTD(1)),
    `block_total_bytes_compressed` Nullable(UInt32) COMMENT 'The total bytes of the beacon block payload when compressed using snappy' CODEC(ZSTD(1)),
    `parent_root` FixedString(66) COMMENT 'The root hash of the parent beacon block' CODEC(ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'The root hash of the beacon state at this block' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The index of the validator that proposed the beacon block' CODEC(ZSTD(1)),
    `eth1_data_block_hash` FixedString(66) COMMENT 'The block hash of the associated execution block' CODEC(ZSTD(1)),
    `eth1_data_deposit_root` FixedString(66) COMMENT 'The root of the deposit tree in the associated execution block' CODEC(ZSTD(1)),
    `execution_payload_block_hash` FixedString(66) COMMENT 'The block hash of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_block_number` UInt32 COMMENT 'The block number of the execution payload' CODEC(DoubleDelta, ZSTD(1)),
    `execution_payload_fee_recipient` String COMMENT 'The recipient of the fee for this execution payload' CODEC(ZSTD(1)),
    `execution_payload_state_root` FixedString(66) COMMENT 'The state root of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_parent_hash` FixedString(66) COMMENT 'The parent hash of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_transactions_count` Nullable(UInt32) COMMENT 'The transaction count of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_transactions_total_bytes` Nullable(UInt32) COMMENT 'The transaction total bytes of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_transactions_total_bytes_compressed` Nullable(UInt32) COMMENT 'The transaction total bytes of the execution payload when compressed using snappy' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains beacon block from a beacon node.';

CREATE TABLE tmp.canonical_beacon_block ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    block_total_bytes,
    block_total_bytes_compressed,
    parent_root,
    state_root,
    proposer_index,
    eth1_data_block_hash,
    eth1_data_deposit_root,
    execution_payload_block_hash,
    execution_payload_block_number,
    execution_payload_fee_recipient,
    execution_payload_state_root,
    execution_payload_parent_hash,
    execution_payload_transactions_count,
    execution_payload_transactions_total_bytes,
    execution_payload_transactions_total_bytes_compressed,
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
    default.canonical_beacon_block_local;

DROP TABLE IF EXISTS default.canonical_beacon_block ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_local
AND tmp.canonical_beacon_block_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block ON CLUSTER '{cluster}' AS default.canonical_beacon_block_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_proposer_slashing
CREATE TABLE tmp.canonical_beacon_block_proposer_slashing_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `signed_header_1_message_slot` UInt32 COMMENT 'The slot number from the first signed header in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `signed_header_1_message_proposer_index` UInt32 COMMENT 'The proposer index from the first signed header in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `signed_header_1_message_body_root` FixedString(66) COMMENT 'The body root from the first signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_1_message_parent_root` FixedString(66) COMMENT 'The parent root from the first signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_1_message_state_root` FixedString(66) COMMENT 'The state root from the first signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_1_signature` String COMMENT 'The signature for the first signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_2_message_slot` UInt32 COMMENT 'The slot number from the second signed header in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `signed_header_2_message_proposer_index` UInt32 COMMENT 'The proposer index from the second signed header in the slashing payload' CODEC(DoubleDelta, ZSTD(1)),
    `signed_header_2_message_body_root` FixedString(66) COMMENT 'The body root from the second signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_2_message_parent_root` FixedString(66) COMMENT 'The parent root from the second signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_2_message_state_root` FixedString(66) COMMENT 'The state root from the second signed header in the slashing payload' CODEC(ZSTD(1)),
    `signed_header_2_signature` String COMMENT 'The signature for the second signed header in the slashing payload' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains proposer slashing from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_proposer_slashing ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_proposer_slashing_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_proposer_slashing_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_proposer_slashing
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            signed_header_1_message_slot,
            signed_header_2_message_slot,
            signed_header_1_message_proposer_index,
            signed_header_2_message_proposer_index,
            signed_header_1_message_body_root,
            signed_header_2_message_body_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    signed_header_1_message_slot,
    signed_header_1_message_proposer_index,
    signed_header_1_message_body_root,
    signed_header_1_message_parent_root,
    signed_header_1_message_state_root,
    signed_header_1_signature,
    signed_header_2_message_slot,
    signed_header_2_message_proposer_index,
    signed_header_2_message_body_root,
    signed_header_2_message_parent_root,
    signed_header_2_message_state_root,
    signed_header_2_signature,
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
    default.canonical_beacon_block_proposer_slashing_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_proposer_slashing ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_proposer_slashing_local
AND tmp.canonical_beacon_block_proposer_slashing_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_proposer_slashing ON CLUSTER '{cluster}' AS default.canonical_beacon_block_proposer_slashing_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_proposer_slashing_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_proposer_slashing ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_proposer_slashing_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_voluntary_exit
CREATE TABLE tmp.canonical_beacon_block_voluntary_exit_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `voluntary_exit_message_epoch` UInt32 COMMENT 'The epoch number from the exit message' CODEC(DoubleDelta, ZSTD(1)),
    `voluntary_exit_message_validator_index` UInt32 COMMENT 'The validator index from the exit message' CODEC(ZSTD(1)),
    `voluntary_exit_signature` String COMMENT 'The signature of the exit message' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains a voluntary exit from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_voluntary_exit_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_voluntary_exit_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_voluntary_exit
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            voluntary_exit_message_epoch,
            voluntary_exit_message_validator_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    voluntary_exit_message_epoch,
    voluntary_exit_message_validator_index,
    voluntary_exit_signature,
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
    default.canonical_beacon_block_voluntary_exit_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_voluntary_exit_local
AND tmp.canonical_beacon_block_voluntary_exit_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}' AS default.canonical_beacon_block_voluntary_exit_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_voluntary_exit_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_voluntary_exit_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_block_withdrawal
CREATE TABLE tmp.canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `withdrawal_index` UInt32 COMMENT 'The index of the withdrawal' CODEC(ZSTD(1)),
    `withdrawal_validator_index` UInt32 COMMENT 'The validator index from the withdrawal data' CODEC(ZSTD(1)),
    `withdrawal_address` FixedString(42) COMMENT 'The address of the account that is the withdrawal recipient' CODEC(ZSTD(1)),
    `withdrawal_amount` UInt128 COMMENT 'The amount of the withdrawal from the withdrawal data' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains a withdrawal from a beacon block.';

CREATE TABLE tmp.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}' AS tmp.canonical_beacon_block_withdrawal_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_block_withdrawal_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_block_withdrawal
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            withdrawal_index,
            withdrawal_validator_index
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    block_root,
    block_version,
    withdrawal_index,
    withdrawal_validator_index,
    withdrawal_address,
    withdrawal_amount,
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
    default.canonical_beacon_block_withdrawal_local;

DROP TABLE IF EXISTS default.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_block_withdrawal_local
AND tmp.canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}' AS default.canonical_beacon_block_withdrawal_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_block_withdrawal_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_elaborated_attestation
CREATE TABLE tmp.canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the elaborated attestation from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `block_slot` UInt32 COMMENT 'The slot number of the block containing the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `block_slot_start_date_time` DateTime COMMENT 'The wall clock time when the block slot started' CODEC(DoubleDelta, ZSTD(1)),
    `block_epoch` UInt32 COMMENT 'The epoch number of the block containing the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `block_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the block epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `position_in_block` UInt32 COMMENT 'The position of the attestation in the block' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root of the block containing the attestation' CODEC(ZSTD(1)),
    `validators` Array(UInt32) COMMENT 'Array of validator indices participating in the attestation' CODEC(ZSTD(1)),
    `committee_index` LowCardinality(String) COMMENT 'The index of the committee making the attestation',
    `beacon_block_root` FixedString(66) COMMENT 'The root of the beacon block being attested to' CODEC(ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number being attested to' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime CODEC(DoubleDelta, ZSTD(1)),
    `source_epoch` UInt32 COMMENT 'The source epoch referenced in the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `source_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the source epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `source_root` FixedString(66) COMMENT 'The root of the source checkpoint in the attestation' CODEC(ZSTD(1)),
    `target_epoch` UInt32 COMMENT 'The target epoch referenced in the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `target_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the target epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `target_root` FixedString(66) COMMENT 'The root of the target checkpoint in the attestation' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains elaborated attestations from beacon blocks.';

CREATE TABLE tmp.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}' AS tmp.canonical_beacon_elaborated_attestation_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_elaborated_attestation_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_elaborated_attestation
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            block_root,
            block_slot,
            position_in_block,
            beacon_block_root,
            slot,
            committee_index,
            source_root,
            target_root
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    block_slot,
    block_slot_start_date_time,
    block_epoch,
    block_epoch_start_date_time,
    position_in_block,
    block_root,
    validators,
    committee_index,
    beacon_block_root,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    source_epoch,
    source_epoch_start_date_time,
    source_root,
    target_epoch,
    target_epoch_start_date_time,
    target_root,
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
    default.canonical_beacon_elaborated_attestation_local;

DROP TABLE IF EXISTS default.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_elaborated_attestation_local
AND tmp.canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}' AS default.canonical_beacon_elaborated_attestation_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_elaborated_attestation_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}' SYNC;

-- canonical_beacon_proposer_duty
CREATE TABLE tmp.canonical_beacon_proposer_duty_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the proposer duty information from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number for which the proposer duty is assigned' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number containing the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `proposer_validator_index` UInt32 COMMENT 'The validator index of the proposer for the slot' CODEC(ZSTD(1)),
    `proposer_pubkey` String COMMENT 'The public key of the validator proposer' CODEC(ZSTD(1)),
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
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the even' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name
    ) COMMENT 'Contains a proposer duty from a beacon block.';

CREATE TABLE tmp.canonical_beacon_proposer_duty ON CLUSTER '{cluster}' AS tmp.canonical_beacon_proposer_duty_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    canonical_beacon_proposer_duty_local,
    unique_key
);

INSERT INTO
    tmp.canonical_beacon_proposer_duty
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            proposer_validator_index,
            proposer_pubkey
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    proposer_validator_index,
    proposer_pubkey,
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
    default.canonical_beacon_proposer_duty_local;

DROP TABLE IF EXISTS default.canonical_beacon_proposer_duty ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.canonical_beacon_proposer_duty_local
AND tmp.canonical_beacon_proposer_duty_local ON CLUSTER '{cluster}';

CREATE TABLE default.canonical_beacon_proposer_duty ON CLUSTER '{cluster}' AS default.canonical_beacon_proposer_duty_local ENGINE = Distributed(
    '{cluster}',
    default,
    canonical_beacon_proposer_duty_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.canonical_beacon_proposer_duty ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.canonical_beacon_proposer_duty_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_add_peer
CREATE TABLE tmp.libp2p_add_peer_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
    `protocol` LowCardinality(String) COMMENT 'Protocol used by the peer',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the details of the peers added to the libp2p client.';

CREATE TABLE tmp.libp2p_add_peer ON CLUSTER '{cluster}' AS tmp.libp2p_add_peer_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_add_peer_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_add_peer
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    peer_id_unique_key,
    protocol,
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
    meta_network_name
FROM
    default.libp2p_add_peer_local;

DROP TABLE IF EXISTS default.libp2p_add_peer ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_add_peer_local
AND tmp.libp2p_add_peer_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_add_peer ON CLUSTER '{cluster}' AS default.libp2p_add_peer_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_add_peer_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_add_peer ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_add_peer_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_connected
CREATE TABLE tmp.libp2p_connected_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `remote_peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the remote peer',
    `remote_protocol` LowCardinality(String) COMMENT 'Protocol of the remote peer',
    `remote_transport_protocol` LowCardinality(String) COMMENT 'Transport protocol of the remote peer',
    `remote_port` UInt16 COMMENT 'Port of the remote peer' CODEC(ZSTD(1)),
    `remote_ip` Nullable(IPv6) COMMENT 'IP address of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_city` LowCardinality(String) COMMENT 'City of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_country` LowCardinality(String) COMMENT 'Country of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_country_code` LowCardinality(String) COMMENT 'Country code of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_agent_implementation` LowCardinality(String) COMMENT 'Implementation of the remote peer',
    `remote_agent_version` LowCardinality(String) COMMENT 'Version of the remote peer',
    `remote_agent_version_major` LowCardinality(String) COMMENT 'Major version of the remote peer',
    `remote_agent_version_minor` LowCardinality(String) COMMENT 'Minor version of the remote peer',
    `remote_agent_version_patch` LowCardinality(String) COMMENT 'Patch version of the remote peer',
    `remote_agent_platform` LowCardinality(String) COMMENT 'Platform of the remote peer',
    `direction` LowCardinality(String) COMMENT 'Connection direction',
    `opened` DateTime COMMENT 'Timestamp when the connection was opened' CODEC(DoubleDelta, ZSTD(1)),
    `transient` Bool COMMENT 'Whether the connection is transient',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the details of the CONNECTED events from the libp2p client.';

CREATE TABLE tmp.libp2p_connected ON CLUSTER '{cluster}' AS tmp.libp2p_connected_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_connected_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_connected
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            remote_peer_id_unique_key,
            direction,
            opened
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    remote_peer_id_unique_key,
    remote_protocol,
    remote_transport_protocol,
    remote_port,
    remote_ip,
    remote_geo_city,
    remote_geo_country,
    remote_geo_country_code,
    remote_geo_continent_code,
    remote_geo_longitude,
    remote_geo_latitude,
    remote_geo_autonomous_system_number,
    remote_geo_autonomous_system_organization,
    remote_agent_implementation,
    remote_agent_version,
    remote_agent_version_major,
    remote_agent_version_minor,
    remote_agent_version_patch,
    remote_agent_platform,
    direction,
    opened,
    transient,
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
    meta_network_name
FROM
    default.libp2p_connected_local;

DROP TABLE IF EXISTS default.libp2p_connected ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_connected_local
AND tmp.libp2p_connected_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_connected ON CLUSTER '{cluster}' AS default.libp2p_connected_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_connected_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_connected ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_connected_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_disconnected
CREATE TABLE tmp.libp2p_disconnected_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `remote_peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the remote peer',
    `remote_protocol` LowCardinality(String) COMMENT 'Protocol of the remote peer',
    `remote_transport_protocol` LowCardinality(String) COMMENT 'Transport protocol of the remote peer',
    `remote_port` UInt16 COMMENT 'Port of the remote peer' CODEC(ZSTD(1)),
    `remote_ip` Nullable(IPv6) COMMENT 'IP address of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_city` LowCardinality(String) COMMENT 'City of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_country` LowCardinality(String) COMMENT 'Country of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_country_code` LowCardinality(String) COMMENT 'Country code of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the remote peer that generated the event' CODEC(ZSTD(1)),
    `remote_agent_implementation` LowCardinality(String) COMMENT 'Implementation of the remote peer',
    `remote_agent_version` LowCardinality(String) COMMENT 'Version of the remote peer',
    `remote_agent_version_major` LowCardinality(String) COMMENT 'Major version of the remote peer',
    `remote_agent_version_minor` LowCardinality(String) COMMENT 'Minor version of the remote peer',
    `remote_agent_version_patch` LowCardinality(String) COMMENT 'Patch version of the remote peer',
    `remote_agent_platform` LowCardinality(String) COMMENT 'Platform of the remote peer',
    `direction` LowCardinality(String) COMMENT 'Connection direction',
    `opened` DateTime COMMENT 'Timestamp when the connection was opened' CODEC(DoubleDelta, ZSTD(1)),
    `transient` Bool COMMENT 'Whether the connection is transient',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the details of the DISCONNECTED events from the libp2p client.';

CREATE TABLE tmp.libp2p_disconnected ON CLUSTER '{cluster}' AS tmp.libp2p_disconnected_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_disconnected_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_disconnected
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            remote_peer_id_unique_key,
            direction,
            opened
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    remote_peer_id_unique_key,
    remote_protocol,
    remote_transport_protocol,
    remote_port,
    remote_ip,
    remote_geo_city,
    remote_geo_country,
    remote_geo_country_code,
    remote_geo_continent_code,
    remote_geo_longitude,
    remote_geo_latitude,
    remote_geo_autonomous_system_number,
    remote_geo_autonomous_system_organization,
    remote_agent_implementation,
    remote_agent_version,
    remote_agent_version_major,
    remote_agent_version_minor,
    remote_agent_version_patch,
    remote_agent_platform,
    direction,
    opened,
    transient,
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
    meta_network_name
FROM
    default.libp2p_disconnected_local;

DROP TABLE IF EXISTS default.libp2p_disconnected ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_disconnected_local
AND tmp.libp2p_disconnected_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_disconnected ON CLUSTER '{cluster}' AS default.libp2p_disconnected_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_disconnected_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_disconnected ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_disconnected_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_gossipsub_beacon_attestation
CREATE TABLE tmp.libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number in the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `committee_index` LowCardinality(String) COMMENT 'The committee index in the attestation',
    `attesting_validator_index` Nullable(UInt32) COMMENT 'The index of the validator attesting to the event' CODEC(ZSTD(1)),
    `attesting_validator_committee_index` LowCardinality(String) COMMENT 'The committee index of the attesting validator',
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference in slot start time for propagation' CODEC(ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
    `message_id` String COMMENT 'Identifier of the message' CODEC(ZSTD(1)),
    `message_size` UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
    `aggregation_bits` String COMMENT 'The aggregation bits of the event in the attestation' CODEC(ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'The beacon block root hash in the attestation' CODEC(ZSTD(1)),
    `source_epoch` UInt32 COMMENT 'The source epoch number in the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `source_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the source epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `source_root` FixedString(66) COMMENT 'The source beacon block root hash in the attestation' CODEC(ZSTD(1)),
    `target_epoch` UInt32 COMMENT 'The target epoch number in the attestation' CODEC(DoubleDelta, ZSTD(1)),
    `target_epoch_start_date_time` DateTime COMMENT 'The wall clock time when the target epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `target_root` FixedString(66) COMMENT 'The target beacon block root hash in the attestation' CODEC(ZSTD(1)),
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
    `meta_network_id` Int32 COMMENT 'Network ID associated with the client' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Table for libp2p gossipsub beacon attestation data.';

CREATE TABLE tmp.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}' AS tmp.libp2p_gossipsub_beacon_attestation_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_gossipsub_beacon_attestation_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_gossipsub_beacon_attestation
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key,
            message_id
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    committee_index,
    attesting_validator_index,
    attesting_validator_committee_index,
    wallclock_slot,
    wallclock_slot_start_date_time,
    wallclock_epoch,
    wallclock_epoch_start_date_time,
    propagation_slot_start_diff,
    peer_id_unique_key,
    message_id,
    message_size,
    topic_layer,
    topic_fork_digest_value,
    topic_name,
    topic_encoding,
    aggregation_bits,
    beacon_block_root,
    source_epoch,
    source_epoch_start_date_time,
    source_root,
    target_epoch,
    target_epoch_start_date_time,
    target_root,
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
    meta_network_name
FROM
    default.libp2p_gossipsub_beacon_attestation_local;

DROP TABLE IF EXISTS default.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_gossipsub_beacon_attestation_local
AND tmp.libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}' AS default.libp2p_gossipsub_beacon_attestation_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_gossipsub_beacon_attestation_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_gossipsub_beacon_block
CREATE TABLE tmp.libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference in slot start time for propagation' CODEC(ZSTD(1)),
    `block` FixedString(66) COMMENT 'The beacon block root hash' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The proposer index of the beacon block' CODEC(ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
    `message_id` String COMMENT 'Identifier of the message' CODEC(ZSTD(1)),
    `message_size` UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
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
    `meta_network_id` Int32 COMMENT 'Network ID associated with the client' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Table for libp2p gossipsub beacon block data.';

CREATE TABLE tmp.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}' AS tmp.libp2p_gossipsub_beacon_block_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_gossipsub_beacon_block_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_gossipsub_beacon_block
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key,
            message_id
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    wallclock_slot,
    wallclock_slot_start_date_time,
    wallclock_epoch,
    wallclock_epoch_start_date_time,
    propagation_slot_start_diff,
    block,
    proposer_index,
    peer_id_unique_key,
    message_id,
    message_size,
    topic_layer,
    topic_fork_digest_value,
    topic_name,
    topic_encoding,
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
    meta_network_name
FROM
    default.libp2p_gossipsub_beacon_block_local;

DROP TABLE IF EXISTS default.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_gossipsub_beacon_block_local
AND tmp.libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}' AS default.libp2p_gossipsub_beacon_block_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_gossipsub_beacon_block_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_gossipsub_blob_sidecar
CREATE TABLE tmp.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference in slot start time for propagation' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The proposer index of the beacon block' CODEC(ZSTD(1)),
    `blob_index` UInt32 COMMENT 'Blob index associated with the record' CODEC(ZSTD(1)),
    `parent_root` FixedString(66) COMMENT 'Parent root of the beacon block' CODEC(ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'State root of the beacon block' CODEC(ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
    `message_id` String COMMENT 'Identifier of the message' CODEC(ZSTD(1)),
    `message_size` UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
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
    `meta_network_id` Int32 COMMENT 'Network ID associated with the client' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Table for libp2p gossipsub blob sidecar data';

CREATE TABLE tmp.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}' AS tmp.libp2p_gossipsub_blob_sidecar_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_gossipsub_blob_sidecar_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_gossipsub_blob_sidecar
SELECT
    toInt64(
        cityHash64(
            slot_start_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key,
            message_id
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    slot,
    slot_start_date_time,
    epoch,
    epoch_start_date_time,
    wallclock_slot,
    wallclock_slot_start_date_time,
    wallclock_epoch,
    wallclock_epoch_start_date_time,
    propagation_slot_start_diff,
    proposer_index,
    blob_index,
    parent_root,
    state_root,
    peer_id_unique_key,
    message_id,
    message_size,
    topic_layer,
    topic_fork_digest_value,
    topic_name,
    topic_encoding,
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
    meta_network_name
FROM
    default.libp2p_gossipsub_blob_sidecar_local;

DROP TABLE IF EXISTS default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_gossipsub_blob_sidecar_local
AND tmp.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}' AS default.libp2p_gossipsub_blob_sidecar_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_gossipsub_blob_sidecar_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_handle_metadata
CREATE TABLE tmp.libp2p_handle_metadata_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer involved in the RPC',
    `error` Nullable(String) COMMENT 'Error message if the metadata handling failed' CODEC(ZSTD(1)),
    `protocol` LowCardinality(String) COMMENT 'The protocol of the metadata handling event',
    `attnets` String COMMENT 'Attestation subnets the peer is subscribed to' CODEC(ZSTD(1)),
    `seq_number` UInt64 COMMENT 'Sequence number of the metadata' CODEC(DoubleDelta, ZSTD(1)),
    `syncnets` String COMMENT 'Sync subnets the peer is subscribed to' CODEC(ZSTD(1)),
    `latency_milliseconds` Decimal(10, 3) COMMENT 'How long it took to handle the metadata request in milliseconds' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the metadata handling events for libp2p peers.';

CREATE TABLE tmp.libp2p_handle_metadata ON CLUSTER '{cluster}' AS tmp.libp2p_handle_metadata_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_handle_metadata_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_handle_metadata
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key,
            seq_number
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    peer_id_unique_key,
    error,
    protocol,
    attnets,
    seq_number,
    syncnets,
    latency_milliseconds,
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
    meta_network_name
FROM
    default.libp2p_handle_metadata_local;

DROP TABLE IF EXISTS default.libp2p_handle_metadata ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_handle_metadata_local
AND tmp.libp2p_handle_metadata_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_handle_metadata ON CLUSTER '{cluster}' AS default.libp2p_handle_metadata_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_handle_metadata_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_handle_metadata ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_handle_metadata_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_handle_status
CREATE TABLE tmp.libp2p_handle_status_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
    `error` Nullable(String) COMMENT 'Error message if the status handling failed' CODEC(ZSTD(1)),
    `protocol` LowCardinality(String) COMMENT 'The protocol of the status handling event',
    `request_finalized_epoch` Nullable(UInt32) COMMENT 'Requested finalized epoch' CODEC(DoubleDelta, ZSTD(1)),
    `request_finalized_root` Nullable(String) COMMENT 'Requested finalized root',
    `request_fork_digest` LowCardinality(String) COMMENT 'Requested fork digest',
    `request_head_root` Nullable(FixedString(66)) COMMENT 'Requested head root' CODEC(ZSTD(1)),
    `request_head_slot` Nullable(UInt32) COMMENT 'Requested head slot' CODEC(ZSTD(1)),
    `response_finalized_epoch` Nullable(UInt32) COMMENT 'Response finalized epoch' CODEC(DoubleDelta, ZSTD(1)),
    `response_finalized_root` Nullable(FixedString(66)) COMMENT 'Response finalized root' CODEC(ZSTD(1)),
    `response_fork_digest` LowCardinality(String) COMMENT 'Response fork digest',
    `response_head_root` Nullable(FixedString(66)) COMMENT 'Response head root' CODEC(ZSTD(1)),
    `response_head_slot` Nullable(UInt32) COMMENT 'Response head slot' CODEC(DoubleDelta, ZSTD(1)),
    `latency_milliseconds` Decimal(10, 3) COMMENT 'How long it took to handle the status request in milliseconds' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the status handling events for libp2p peers.';

CREATE TABLE tmp.libp2p_handle_status ON CLUSTER '{cluster}' AS tmp.libp2p_handle_status_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_handle_status_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_handle_status
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    peer_id_unique_key,
    error,
    protocol,
    request_finalized_epoch,
    request_finalized_root,
    request_fork_digest,
    request_head_root,
    request_head_slot,
    response_finalized_epoch,
    response_finalized_root,
    response_fork_digest,
    response_head_root,
    response_head_slot,
    latency_milliseconds,
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
    meta_network_name
FROM
    default.libp2p_handle_status_local;

DROP TABLE IF EXISTS default.libp2p_handle_status ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_handle_status_local
AND tmp.libp2p_handle_status_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_handle_status ON CLUSTER '{cluster}' AS default.libp2p_handle_status_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_handle_status_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_handle_status ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_handle_status_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_join
CREATE TABLE tmp.libp2p_join_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding of the topic',
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer that joined the topic',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the details of the JOIN events from the libp2p client.';

CREATE TABLE tmp.libp2p_join ON CLUSTER '{cluster}' AS tmp.libp2p_join_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_join_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_join
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key,
            topic_fork_digest_value,
            topic_name
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    topic_layer,
    topic_fork_digest_value,
    topic_name,
    topic_encoding,
    peer_id_unique_key,
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
    meta_network_name
FROM
    default.libp2p_join_local;

DROP TABLE IF EXISTS default.libp2p_join ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_join_local
AND tmp.libp2p_join_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_join ON CLUSTER '{cluster}' AS default.libp2p_join_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_join_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_join ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_join_local ON CLUSTER '{cluster}' SYNC;

-- libp2p_remove_peer
CREATE TABLE tmp.libp2p_remove_peer_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Contains the details of the peers removed from the libp2p client.';

CREATE TABLE tmp.libp2p_remove_peer ON CLUSTER '{cluster}' AS tmp.libp2p_remove_peer_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    libp2p_remove_peer_local,
    unique_key
);

INSERT INTO
    tmp.libp2p_remove_peer
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            peer_id_unique_key
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    peer_id_unique_key,
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
    meta_network_name
FROM
    default.libp2p_remove_peer_local;

DROP TABLE IF EXISTS default.libp2p_remove_peer ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.libp2p_remove_peer_local
AND tmp.libp2p_remove_peer_local ON CLUSTER '{cluster}';

CREATE TABLE default.libp2p_remove_peer ON CLUSTER '{cluster}' AS default.libp2p_remove_peer_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_remove_peer_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.libp2p_remove_peer ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.libp2p_remove_peer_local ON CLUSTER '{cluster}' SYNC;

-- mempool_dumpster_transaction
CREATE TABLE tmp.mempool_dumpster_transaction_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'When this row was last updated, this is outside the source data and used for deduplication' CODEC(DoubleDelta, ZSTD(1)),
    `timestamp` DateTime64(3) COMMENT 'Timestamp of the transaction' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'The hash of the transaction' CODEC(ZSTD(1)),
    `chain_id` UInt32 COMMENT 'The chain id of the transaction' CODEC(ZSTD(1)),
    `from` FixedString(42) COMMENT 'The address of the account that sent the transaction' CODEC(ZSTD(1)),
    `to` Nullable(FixedString(42)) COMMENT 'The address of the account that is the transaction recipient' CODEC(ZSTD(1)),
    `value` UInt128 COMMENT 'The value transferred with the transaction in wei' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The nonce of the sender account at the time of the transaction' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'The maximum gas provided for the transaction execution' CODEC(ZSTD(1)),
    `gas_price` UInt128 COMMENT 'The gas price of the transaction in wei' CODEC(ZSTD(1)),
    `gas_tip_cap` Nullable(UInt128) COMMENT 'The gas tip cap of the transaction in wei' CODEC(ZSTD(1)),
    `gas_fee_cap` Nullable(UInt128) COMMENT 'The gas fee cap of the transaction in wei' CODEC(ZSTD(1)),
    `data_size` UInt32 COMMENT 'The size of the call data of the transaction in bytes' CODEC(ZSTD(1)),
    `data_4bytes` Nullable(FixedString(10)) COMMENT 'The first 4 bytes of the call data of the transaction' CODEC(ZSTD(1)),
    `sources` Array(LowCardinality(String)) COMMENT 'The sources that saw this transaction in their mempool',
    `included_at_block_height` Nullable(UInt64) COMMENT 'The block height at which this transaction was included' CODEC(ZSTD(1)),
    `included_block_timestamp` Nullable(DateTime64(3)) COMMENT 'The timestamp of the block at which this transaction was included' CODEC(DoubleDelta, ZSTD(1)),
    `inclusion_delay_ms` Nullable(Int64) COMMENT 'The delay between the transaction timestamp and the block timestamp' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(timestamp)
ORDER BY
    (
        timestamp,
        unique_key,
        chain_id
    ) COMMENT 'Contains transactions from mempool dumpster dataset. Following the parquet schema with some additions';

CREATE TABLE tmp.mempool_dumpster_transaction ON CLUSTER '{cluster}' AS tmp.mempool_dumpster_transaction_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    mempool_dumpster_transaction_local,
    unique_key
);

INSERT INTO
    tmp.mempool_dumpster_transaction
SELECT
    toInt64(
        cityHash64(
            timestamp,
            chain_id,
            `from`,
            nonce,
            gas
        ) - 9223372036854775808
    ),
    NOW(),
    timestamp,
    hash,
    chain_id,
    `from`,
    to,
    value,
    nonce,
    gas,
    gas_price,
    gas_tip_cap,
    gas_fee_cap,
    data_size,
    data_4bytes,
    sources,
    included_at_block_height,
    included_block_timestamp,
    inclusion_delay_ms
FROM
    default.mempool_dumpster_transaction_local;

DROP TABLE IF EXISTS default.mempool_dumpster_transaction ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.mempool_dumpster_transaction_local
AND tmp.mempool_dumpster_transaction_local ON CLUSTER '{cluster}';

CREATE TABLE default.mempool_dumpster_transaction ON CLUSTER '{cluster}' AS default.mempool_dumpster_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    mempool_dumpster_transaction_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.mempool_dumpster_transaction ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.mempool_dumpster_transaction_local ON CLUSTER '{cluster}' SYNC;

-- mempool_transaction
CREATE TABLE tmp.mempool_transaction_local ON CLUSTER '{cluster}' (
    `unique_key` Int64 COMMENT 'Unique identifier for each record',
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'The time when the sentry saw the transaction in the mempool' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'The hash of the transaction' CODEC(ZSTD(1)),
    `from` FixedString(42) COMMENT 'The address of the account that sent the transaction' CODEC(ZSTD(1)),
    `to` Nullable(FixedString(42)) COMMENT 'The address of the account that is the transaction recipient' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The nonce of the sender account at the time of the transaction' CODEC(ZSTD(1)),
    `gas_price` UInt128 COMMENT 'The gas price of the transaction in wei' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'The maximum gas provided for the transaction execution' CODEC(ZSTD(1)),
    `gas_tip_cap` Nullable(UInt128) COMMENT 'The priority fee (tip) the user has set for the transaction',
    `gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee the user has set for the transaction',
    `value` UInt128 COMMENT 'The value transferred with the transaction in wei' CODEC(ZSTD(1)),
    `type` Nullable(UInt8) COMMENT 'The type of the transaction',
    `size` UInt32 COMMENT 'The size of the transaction data in bytes' CODEC(ZSTD(1)),
    `call_data_size` UInt32 COMMENT 'The size of the call data of the transaction in bytes' CODEC(ZSTD(1)),
    `blob_gas` Nullable(UInt64) COMMENT 'The maximum gas provided for the blob transaction execution',
    `blob_gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee the user has set for the transaction',
    `blob_hashes` Array(String) COMMENT 'The hashes of the blob commitments for blob transactions',
    `blob_sidecars_size` Nullable(UInt32) COMMENT 'The total size of the sidecars for blob transactions in bytes',
    `blob_sidecars_empty_size` Nullable(UInt32) COMMENT 'The total empty size of the sidecars for blob transactions in bytes',
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
    `meta_execution_fork_id_hash` LowCardinality(String) COMMENT 'The hash of the fork ID of the current Ethereum network',
    `meta_execution_fork_id_next` LowCardinality(String) COMMENT 'The fork ID of the next planned Ethereum network upgrade',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(event_date_time)
ORDER BY
    (
        event_date_time,
        unique_key,
        meta_network_name,
        meta_client_name
    ) COMMENT 'Each row represents a transaction that was seen in the mempool by a sentry client. Sentries can report the same transaction multiple times if it has been long enough since the last report.';

CREATE TABLE tmp.mempool_transaction ON CLUSTER '{cluster}' AS tmp.mempool_transaction_local ENGINE = Distributed(
    '{cluster}',
    tmp,
    mempool_transaction_local,
    unique_key
);

INSERT INTO
    tmp.mempool_transaction
SELECT
    toInt64(
        cityHash64(
            event_date_time,
            meta_network_name,
            meta_client_name,
            hash,
            `from`,
            nonce,
            gas
        ) - 9223372036854775808
    ),
    NOW(),
    event_date_time,
    hash,
    `from`,
    to,
    nonce,
    gas_price,
    gas,
    gas_tip_cap,
    gas_fee_cap,
    value,
    type,
    size,
    call_data_size,
    blob_gas,
    blob_gas_fee_cap,
    blob_hashes,
    blob_sidecars_size,
    blob_sidecars_empty_size,
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
    meta_execution_fork_id_hash,
    meta_execution_fork_id_next,
    meta_labels
FROM
    default.mempool_transaction_local;

DROP TABLE IF EXISTS default.mempool_transaction ON CLUSTER '{cluster}' SYNC;

EXCHANGE TABLES default.mempool_transaction_local
AND tmp.mempool_transaction_local ON CLUSTER '{cluster}';

CREATE TABLE default.mempool_transaction ON CLUSTER '{cluster}' AS default.mempool_transaction_local ENGINE = Distributed(
    '{cluster}',
    default,
    mempool_transaction_local,
    unique_key
);

DROP TABLE IF EXISTS tmp.mempool_transaction ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS tmp.mempool_transaction_local ON CLUSTER '{cluster}' SYNC;