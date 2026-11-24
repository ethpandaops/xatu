-- Cannon: per-block graffiti.

CREATE TABLE IF NOT EXISTS canonical_beacon_block_graffiti_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `proposer_index` UInt32 COMMENT 'The index of the validator that proposed the beacon block' CODEC(ZSTD(1)),
    `graffiti` FixedString(66) COMMENT 'The raw 32 byte graffiti from the beacon block body (hex encoded)' CODEC(ZSTD(1)),
    `graffiti_text` String COMMENT 'The graffiti decoded as UTF-8 with trailing null bytes removed and invalid sequences replaced' CODEC(ZSTD(1)),
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
COMMENT 'Contains the graffiti from a beacon block (1 row per block).';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_graffiti ON CLUSTER '{cluster}'
AS canonical_beacon_block_graffiti_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_graffiti_local, cityHash64(slot_start_date_time, meta_network_name, block_root))
COMMENT 'Contains the graffiti from a beacon block (1 row per block).';
