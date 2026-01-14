CREATE TABLE default.execution_mpt_depth_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'State root hash of the execution layer at this block' CODEC(ZSTD(1)),
    `parent_state_root` FixedString(66) COMMENT 'State root hash of the execution layer at the parent block' CODEC(ZSTD(1)),
    -- Total metrics (fast access for common queries)
    `total_account_written_nodes` UInt64 COMMENT 'The total number of trie nodes written at all depths of account tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_account_written_bytes` UInt64 COMMENT 'The total number of bytes written at all depths of account tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_account_deleted_nodes` UInt64 COMMENT 'The total number of trie nodes deleted at all depths of account tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_account_deleted_bytes` UInt64 COMMENT 'The total number of bytes deleted at all depths of account tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_storage_written_nodes` UInt64 COMMENT 'The total number of trie nodes written at all depths of storage tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_storage_written_bytes` UInt64 COMMENT 'The total number of bytes written at all depths of storage tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_storage_deleted_nodes` UInt64 COMMENT 'The total number of trie nodes deleted at all depths of storage tries' CODEC(DoubleDelta, ZSTD(1)),
    `total_storage_deleted_bytes` UInt64 COMMENT 'The total number of bytes deleted at all depths of storage tries' CODEC(DoubleDelta, ZSTD(1)),
    -- Per-depth metrics using Maps (key: depth 0-64, value: count/bytes)
    `account_written_nodes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of nodes written at that depth of account trie' CODEC(ZSTD(1)),
    `account_written_bytes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of bytes written at that depth of account trie' CODEC(ZSTD(1)),
    `account_deleted_nodes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of nodes deleted at that depth of account trie' CODEC(ZSTD(1)),
    `account_deleted_bytes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of bytes deleted at that depth of account trie' CODEC(ZSTD(1)),
    `storage_written_nodes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of nodes written at that depth of storage tries' CODEC(ZSTD(1)),
    `storage_written_bytes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of bytes written at that depth of storage tries' CODEC(ZSTD(1)),
    `storage_deleted_nodes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of nodes deleted at that depth of storage tries' CODEC(ZSTD(1)),
    `storage_deleted_bytes` Map(UInt8, UInt64) COMMENT 'Map of depth to number of bytes deleted at that depth of storage tries' CODEC(ZSTD(1)),
    -- Standard metadata fields
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
    `meta_execution_version` LowCardinality(String) COMMENT 'Execution client version that generated the event',
    `meta_execution_version_major` LowCardinality(String) COMMENT 'Execution client major version that generated the event',
    `meta_execution_version_minor` LowCardinality(String) COMMENT 'Execution client minor version that generated the event',
    `meta_execution_version_patch` LowCardinality(String) COMMENT 'Execution client patch version that generated the event',
    `meta_execution_implementation` LowCardinality(String) COMMENT 'Execution client implementation that generated the event',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY intDiv(block_number, 5000000)
ORDER BY (
    block_number,
    meta_network_name,
    meta_client_name,
    state_root
) COMMENT 'Contains execution layer Merkle Patricia Trie depth metrics including nodes written and nodes deleted at specific block heights.';

CREATE TABLE default.execution_mpt_depth ON CLUSTER '{cluster}' AS default.execution_mpt_depth_local ENGINE = Distributed(
    '{cluster}',
    default,
    execution_mpt_depth_local,
    cityHash64(
        block_number,
        meta_network_name,
        meta_client_name,
        state_root
    )
);
