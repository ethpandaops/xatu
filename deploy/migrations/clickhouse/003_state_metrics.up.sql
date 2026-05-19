CREATE TABLE IF NOT EXISTS default.execution_state_size_delta_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'State root hash of the execution layer at this block' Codec(ZSTD(1)),
    `parent_state_root` FixedString(66) COMMENT 'State root hash of the execution layer at the parent block' Codec(ZSTD(1)),
    -- Writes: number/bytes of state entries written at this block (updates count here).
    `account_writes` UInt64 COMMENT 'Accounts written at this block (creations + updates)' Codec(DoubleDelta, ZSTD(1)),
    `account_write_bytes` UInt64 COMMENT 'Bytes of account data written at this block' Codec(DoubleDelta, ZSTD(1)),
    `account_trienode_writes` UInt64 COMMENT 'Account trie nodes written at this block' Codec(DoubleDelta, ZSTD(1)),
    `account_trienode_write_bytes` UInt64 COMMENT 'Bytes of account trie node data written at this block' Codec(DoubleDelta, ZSTD(1)),
    `contract_code_writes` UInt64 COMMENT 'New unique contract code blobs added at this block (deduped by hash)' Codec(DoubleDelta, ZSTD(1)),
    `contract_code_write_bytes` UInt64 COMMENT 'Bytes of new contract code added at this block' Codec(DoubleDelta, ZSTD(1)),
    `storage_writes` UInt64 COMMENT 'Storage slots written at this block (creations + updates)' Codec(DoubleDelta, ZSTD(1)),
    `storage_write_bytes` UInt64 COMMENT 'Bytes of storage slot data written at this block' Codec(DoubleDelta, ZSTD(1)),
    `storage_trienode_writes` UInt64 COMMENT 'Storage trie nodes written at this block' Codec(DoubleDelta, ZSTD(1)),
    `storage_trienode_write_bytes` UInt64 COMMENT 'Bytes of storage trie node data written at this block' Codec(DoubleDelta, ZSTD(1)),
    -- Deletes: number/bytes of state entries deleted at this block (updates count here too).
    -- contract_code_deletes / contract_code_delete_bytes are always 0: the geth state sizer
    -- does not reference-count code blobs, so blob-level deletions cannot be safely attributed.
    `account_deletes` UInt64 COMMENT 'Accounts deleted at this block (deletions + updates)' Codec(DoubleDelta, ZSTD(1)),
    `account_delete_bytes` UInt64 COMMENT 'Bytes of account data deleted at this block' Codec(DoubleDelta, ZSTD(1)),
    `account_trienode_deletes` UInt64 COMMENT 'Account trie nodes deleted at this block' Codec(DoubleDelta, ZSTD(1)),
    `account_trienode_delete_bytes` UInt64 COMMENT 'Bytes of account trie node data deleted at this block' Codec(DoubleDelta, ZSTD(1)),
    `contract_code_deletes` UInt64 COMMENT 'Always 0 - reserved for future ref-counted code blob tracking' Codec(DoubleDelta, ZSTD(1)),
    `contract_code_delete_bytes` UInt64 COMMENT 'Always 0 - reserved for future ref-counted code blob tracking' Codec(DoubleDelta, ZSTD(1)),
    `storage_deletes` UInt64 COMMENT 'Storage slots deleted at this block (deletions + updates)' Codec(DoubleDelta, ZSTD(1)),
    `storage_delete_bytes` UInt64 COMMENT 'Bytes of storage slot data deleted at this block' Codec(DoubleDelta, ZSTD(1)),
    `storage_trienode_deletes` UInt64 COMMENT 'Storage trie nodes deleted at this block' Codec(DoubleDelta, ZSTD(1)),
    `storage_trienode_delete_bytes` UInt64 COMMENT 'Bytes of storage trie node data deleted at this block' Codec(DoubleDelta, ZSTD(1)),
    -- Derived net deltas: computed at insert time as (writes - deletes). Inputs are cast to
    -- Int64 first because UInt64 - UInt64 in ClickHouse stays UInt64 (wrapping on underflow).
    `account_delta` Int64 MATERIALIZED toInt64(account_writes) - toInt64(account_deletes) COMMENT 'Net change in accounts (writes - deletes)',
    `account_bytes_delta` Int64 MATERIALIZED toInt64(account_write_bytes) - toInt64(account_delete_bytes) COMMENT 'Net change in account bytes',
    `account_trienode_delta` Int64 MATERIALIZED toInt64(account_trienode_writes) - toInt64(account_trienode_deletes) COMMENT 'Net change in account trie nodes',
    `account_trienode_bytes_delta` Int64 MATERIALIZED toInt64(account_trienode_write_bytes) - toInt64(account_trienode_delete_bytes) COMMENT 'Net change in account trie node bytes',
    `contract_code_delta` Int64 MATERIALIZED toInt64(contract_code_writes) - toInt64(contract_code_deletes) COMMENT 'Net change in contract codes (equals contract_code_writes while deletes are untracked)',
    `contract_code_bytes_delta` Int64 MATERIALIZED toInt64(contract_code_write_bytes) - toInt64(contract_code_delete_bytes) COMMENT 'Net change in contract code bytes',
    `storage_delta` Int64 MATERIALIZED toInt64(storage_writes) - toInt64(storage_deletes) COMMENT 'Net change in storage slots',
    `storage_bytes_delta` Int64 MATERIALIZED toInt64(storage_write_bytes) - toInt64(storage_delete_bytes) COMMENT 'Net change in storage slot bytes',
    `storage_trienode_delta` Int64 MATERIALIZED toInt64(storage_trienode_writes) - toInt64(storage_trienode_deletes) COMMENT 'Net change in storage trie nodes',
    `storage_trienode_bytes_delta` Int64 MATERIALIZED toInt64(storage_trienode_write_bytes) - toInt64(storage_trienode_delete_bytes) COMMENT 'Net change in storage trie node bytes',
    -- Standard metadata fields
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' Codec(ZSTD(1)),
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client that generated the event',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' Codec(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' Codec(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' Codec(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_execution_version` LowCardinality(String) COMMENT 'Execution client version that generated the event',
    `meta_execution_version_major` LowCardinality(String) COMMENT 'Execution client major version that generated the event',
    `meta_execution_version_minor` LowCardinality(String) COMMENT 'Execution client minor version that generated the event',
    `meta_execution_version_patch` LowCardinality(String) COMMENT 'Execution client patch version that generated the event',
    `meta_execution_implementation` LowCardinality(String) COMMENT 'Execution client implementation that generated the event',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' Codec(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY (meta_network_name, intDiv(block_number, 5000000))
ORDER BY (
        meta_network_name,
        block_number,
        meta_client_name,
        state_root
    ) COMMENT 'Contains execution layer state size write/delete metrics (count and bytes for accounts, storage, contract code, and account/storage trie nodes) at specific block heights. Net deltas are MATERIALIZED columns derived as writes - deletes.';
CREATE TABLE IF NOT EXISTS default.execution_state_size_delta ON CLUSTER '{cluster}' AS default.execution_state_size_delta_local ENGINE = Distributed(
    '{cluster}',
    default,
    execution_state_size_delta_local,
    cityHash64(
        block_number,
        meta_network_name,
        meta_client_name,
        state_root
    )
);
CREATE TABLE IF NOT EXISTS default.execution_mpt_depth_local ON CLUSTER '{cluster}' (
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
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    updated_date_time
) PARTITION BY (meta_network_name, intDiv(block_number, 5000000))
ORDER BY (
    meta_network_name,
    block_number,
    meta_client_name,
    state_root
) COMMENT 'Contains execution layer Merkle Patricia Trie depth metrics including nodes written and nodes deleted at specific block heights.';

CREATE TABLE IF NOT EXISTS default.execution_mpt_depth ON CLUSTER '{cluster}' AS default.execution_mpt_depth_local ENGINE = Distributed(
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
