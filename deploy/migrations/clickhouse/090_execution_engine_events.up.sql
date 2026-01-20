CREATE TABLE execution_engine_new_payload_local ON CLUSTER '{cluster}' (
  -- Timestamps
  updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' Codec(DoubleDelta, ZSTD(1)),
  event_date_time DateTime64(3) COMMENT 'When the event was received' Codec(DoubleDelta, ZSTD(1)),
  requested_date_time DateTime64(3) COMMENT 'Timestamp when the engine_newPayload call was received' Codec(DoubleDelta, ZSTD(1)),

  -- Timing
  duration_ms UInt32 COMMENT 'How long the engine_newPayload call took in milliseconds' Codec(ZSTD(1)),

  -- Source
  source LowCardinality(String) COMMENT 'Source of the event (SNOOPER, EXECUTION_CLIENT)',

  -- Execution payload details
  block_number UInt64 COMMENT 'Execution block number' Codec(DoubleDelta, ZSTD(1)),
  block_hash FixedString(66) COMMENT 'Execution block hash (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  parent_hash FixedString(66) COMMENT 'Parent execution block hash (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  gas_used UInt64 COMMENT 'Total gas used by all transactions in the block' Codec(ZSTD(1)),
  gas_limit UInt64 COMMENT 'Gas limit of the block' Codec(ZSTD(1)),
  tx_count UInt32 COMMENT 'Number of transactions in the block' Codec(ZSTD(1)),
  blob_count UInt32 COMMENT 'Number of blobs in the block' Codec(ZSTD(1)),

  -- Response from EL
  status LowCardinality(String) COMMENT 'Payload status returned (VALID, INVALID, SYNCING, ACCEPTED, INVALID_BLOCK_HASH)',
  latest_valid_hash Nullable(FixedString(66)) COMMENT 'Latest valid hash when status is INVALID (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  validation_error Nullable(String) COMMENT 'Error message when validation fails' Codec(ZSTD(1)),

  -- Meta
  method_version LowCardinality(String) COMMENT 'Version of the engine_newPayload method (e.g., V3, V4)',

  -- Execution client metadata
  meta_execution_implementation LowCardinality(String) COMMENT 'Implementation of the execution client (e.g., go-ethereum, reth, nethermind)',
  meta_execution_version LowCardinality(String) COMMENT 'Version of the execution client',
  meta_execution_version_major LowCardinality(String) COMMENT 'Major version number of the execution client',
  meta_execution_version_minor LowCardinality(String) COMMENT 'Minor version number of the execution client',
  meta_execution_version_patch LowCardinality(String) COMMENT 'Patch version number of the execution client',

  -- Standard metadata fields
  meta_client_name LowCardinality(String) COMMENT 'Name of the client that generated the event',
  meta_client_id String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' Codec(ZSTD(1)),
  meta_client_version LowCardinality(String) COMMENT 'Version of the client that generated the event',
  meta_client_implementation LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
  meta_client_os LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
  meta_client_ip Nullable(IPv6) COMMENT 'IP address of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) COMMENT 'City of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) COMMENT 'Country of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) COMMENT 'Country code of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) COMMENT 'Continent code of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) COMMENT 'Longitude of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) COMMENT 'Latitude of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' Codec(ZSTD(1)),
  meta_network_id Int32 COMMENT 'Ethereum network ID' Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY intDiv(block_number, 5000000)
ORDER BY (block_number, meta_network_name, meta_client_name, block_hash, event_date_time) COMMENT 'Contains timing and instrumentation data for engine_newPayload calls from the execution layer perspective.';

CREATE TABLE execution_engine_new_payload ON CLUSTER '{cluster}' AS execution_engine_new_payload_local
ENGINE = Distributed(
  '{cluster}',
  default,
  execution_engine_new_payload_local,
  cityHash64(
    block_number,
    meta_network_name,
    meta_client_name,
    block_hash,
    event_date_time
  )
);

CREATE TABLE execution_engine_get_blobs_local ON CLUSTER '{cluster}' (
  -- Timestamps
  updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' Codec(DoubleDelta, ZSTD(1)),
  event_date_time DateTime64(3) COMMENT 'When the event was received' Codec(DoubleDelta, ZSTD(1)),
  requested_date_time DateTime64(3) COMMENT 'Timestamp when the engine_getBlobs call was received' Codec(DoubleDelta, ZSTD(1)),

  -- Timing
  duration_ms UInt32 COMMENT 'How long the engine_getBlobs call took in milliseconds' Codec(ZSTD(1)),

  -- Source
  source LowCardinality(String) COMMENT 'Source of the event (SNOOPER, EXECUTION_CLIENT)',

  -- Request details
  requested_count UInt32 COMMENT 'Number of versioned hashes requested' Codec(ZSTD(1)),
  versioned_hashes Array(FixedString(66)) COMMENT 'List of versioned hashes requested (hex encoded)' Codec(ZSTD(1)),

  -- Response from EL
  returned_count UInt32 COMMENT 'Number of non-null blobs returned' Codec(ZSTD(1)),
  status LowCardinality(String) COMMENT 'Result status (SUCCESS, PARTIAL, EMPTY, UNSUPPORTED, ERROR)',
  error_message Nullable(String) COMMENT 'Error details if status is ERROR or UNSUPPORTED' Codec(ZSTD(1)),

  -- Meta
  method_version LowCardinality(String) COMMENT 'Version of the engine_getBlobs method (e.g., V1, V2)',

  -- Execution client metadata
  meta_execution_implementation LowCardinality(String) COMMENT 'Implementation of the execution client (e.g., go-ethereum, reth, nethermind)',
  meta_execution_version LowCardinality(String) COMMENT 'Version of the execution client',
  meta_execution_version_major LowCardinality(String) COMMENT 'Major version number of the execution client',
  meta_execution_version_minor LowCardinality(String) COMMENT 'Minor version number of the execution client',
  meta_execution_version_patch LowCardinality(String) COMMENT 'Patch version number of the execution client',

  -- Standard metadata fields
  meta_client_name LowCardinality(String) COMMENT 'Name of the client that generated the event',
  meta_client_id String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' Codec(ZSTD(1)),
  meta_client_version LowCardinality(String) COMMENT 'Version of the client that generated the event',
  meta_client_implementation LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
  meta_client_os LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
  meta_client_ip Nullable(IPv6) COMMENT 'IP address of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) COMMENT 'City of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) COMMENT 'Country of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) COMMENT 'Country code of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) COMMENT 'Continent code of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) COMMENT 'Longitude of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) COMMENT 'Latitude of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' Codec(ZSTD(1)),
  meta_network_id Int32 COMMENT 'Ethereum network ID' Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(event_date_time)
ORDER BY (event_date_time, meta_network_name, meta_client_name) COMMENT 'Contains timing and instrumentation data for engine_getBlobs calls from the execution layer perspective.';

CREATE TABLE execution_engine_get_blobs ON CLUSTER '{cluster}' AS execution_engine_get_blobs_local
ENGINE = Distributed(
  '{cluster}',
  default,
  execution_engine_get_blobs_local,
  cityHash64(
    event_date_time,
    meta_network_name,
    meta_client_name
  )
);
