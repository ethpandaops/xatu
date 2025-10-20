-- Create local table (actual storage)
CREATE TABLE IF NOT EXISTS libp2p_rpc_data_column_custody_probe_local ON CLUSTER '{cluster}' (
    -- Timestamps
    updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) COMMENT 'When the probe was executed' CODEC(DoubleDelta, ZSTD(1)),

    -- Probe identifiers
    slot UInt32 COMMENT 'Slot number being probed' CODEC(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    epoch UInt32 COMMENT 'Epoch number of the slot being probed' CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),

    wallclock_request_slot UInt32 COMMENT 'The wallclock slot when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_request_slot_start_date_time DateTime COMMENT 'The start time for the slot when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_request_epoch UInt32 COMMENT 'The wallclock epoch when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_request_epoch_start_date_time DateTime COMMENT 'The start time for the wallclock epoch when the request was sent' CODEC(DoubleDelta, ZSTD(1)),

    -- Column information
    column_index UInt64 COMMENT 'Column index being probed' CODEC(ZSTD(1)),
    column_rows_count UInt16 COMMENT 'Number of rows in the column' CODEC(ZSTD(1)),
    beacon_block_root FixedString(66) COMMENT 'Root of the beacon block' CODEC(ZSTD(1)),

    -- Peer information
    peer_id_unique_key Int64 COMMENT 'Unique key associated with the identifier of the peer',

    -- Probe results
    result LowCardinality(String) COMMENT 'Result of the probe' CODEC(ZSTD(1)),
    response_time_ms Int32 COMMENT 'Response time in milliseconds' CODEC(ZSTD(1)),
    error Nullable(String) COMMENT 'Error message if probe failed' CODEC(ZSTD(1)),

    -- Standard metadata fields
    meta_client_name LowCardinality(String) COMMENT 'Name of the client that executed the probe',
    meta_client_id String COMMENT 'Unique Session ID of the client' CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String) COMMENT 'Version of the client',
    meta_client_implementation LowCardinality(String) COMMENT 'Implementation of the client',
    meta_client_os LowCardinality(String) COMMENT 'Operating system of the client',
    meta_client_ip Nullable(IPv6) COMMENT 'IP address of the client' CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) COMMENT 'City of the client' CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) COMMENT 'Country of the client' CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) COMMENT 'Country code of the client' CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) COMMENT 'Continent code of the client' CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) COMMENT 'Longitude of the client' CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) COMMENT 'Latitude of the client' CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) COMMENT 'Autonomous system number of the client' CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) COMMENT 'Autonomous system organization of the client' CODEC(ZSTD(1)),
    meta_network_id Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String) COMMENT 'Ethereum network name',
    meta_consensus_version LowCardinality(String) COMMENT 'Ethereum consensus client version',
    meta_consensus_version_major LowCardinality(String) COMMENT 'Ethereum consensus client major version',
    meta_consensus_version_minor LowCardinality(String) COMMENT 'Ethereum consensus client minor version',
    meta_consensus_version_patch LowCardinality(String) COMMENT 'Ethereum consensus client patch version',
    meta_consensus_implementation LowCardinality(String) COMMENT 'Ethereum consensus client implementation',
    meta_labels Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (
    slot_start_date_time,
    meta_network_name,
    meta_client_name,
    peer_id_unique_key,
    slot,
    column_index,
    event_date_time
)
COMMENT 'Contains custody probe events for data column availability verification';

-- Create distributed table (query interface)
CREATE TABLE IF NOT EXISTS libp2p_rpc_data_column_custody_probe ON CLUSTER '{cluster}'
AS libp2p_rpc_data_column_custody_probe_local
ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_rpc_data_column_custody_probe_local,
    cityHash64(
        slot_start_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        slot,
        column_index,
        event_date_time
    )
);
