-- Cannon: Gloas (EIP-7928) per-block block access list summary.

CREATE TABLE IF NOT EXISTS canonical_beacon_block_access_list_summary_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `block_number` UInt64 COMMENT 'The execution block number from the execution payload' CODEC(DoubleDelta, ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'The execution block hash from the execution payload' CODEC(ZSTD(1)),
    `accounts_touched` UInt32 COMMENT 'The number of distinct addresses in the block access list' CODEC(ZSTD(1)),
    `storage_slots_changed` UInt32 COMMENT 'The number of distinct (address, slot) pairs written at least once' CODEC(ZSTD(1)),
    `storage_changes` UInt32 COMMENT 'The number of individual storage write records (one per slot and block access index)' CODEC(ZSTD(1)),
    `storage_reads` UInt32 COMMENT 'The number of distinct (address, slot) pairs that were only read' CODEC(ZSTD(1)),
    `balance_changes` UInt32 COMMENT 'The number of balance change records' CODEC(ZSTD(1)),
    `nonce_changes` UInt32 COMMENT 'The number of nonce change records' CODEC(ZSTD(1)),
    `code_changes` UInt32 COMMENT 'The number of code change records' CODEC(ZSTD(1)),
    `total_changes` UInt32 COMMENT 'The total number of change records: storage_changes + balance_changes + nonce_changes + code_changes' CODEC(ZSTD(1)),
    `bal_size_bytes` UInt32 COMMENT 'The length in bytes of the RLP encoded block access list as carried in the execution payload' CODEC(ZSTD(1)),
    `bal_hash` FixedString(66) COMMENT 'The keccak256 hash of the RLP encoded block access list as carried in the execution payload' CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client' CODEC(ZSTD(1)),
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client' CODEC(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client' CODEC(ZSTD(1)),
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client' CODEC(ZSTD(1)),
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client' CODEC(ZSTD(1)),
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client' CODEC(ZSTD(1)),
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client' CODEC(ZSTD(1)),
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'ASN of the client' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'AS organization of the client' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, block_root)
COMMENT 'Contains a per-block summary of the EIP-7928 block access list from a beacon block (1 row per block).';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_access_list_summary ON CLUSTER '{cluster}'
AS canonical_beacon_block_access_list_summary_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_access_list_summary_local, cityHash64(slot_start_date_time, meta_network_name, block_root))
COMMENT 'Contains a per-block summary of the EIP-7928 block access list from a beacon block (1 row per block).';
