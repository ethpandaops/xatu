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
