CREATE TABLE consensus_engine_api_new_payload_local ON CLUSTER '{cluster}' (
  -- Timestamps
  updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' Codec(DoubleDelta, ZSTD(1)),
  event_date_time DateTime64(3) COMMENT 'When the sentry received the event' Codec(DoubleDelta, ZSTD(1)),
  requested_at DateTime64(3) COMMENT 'Timestamp when the engine_newPayload call was initiated' Codec(DoubleDelta, ZSTD(1)),

  -- Timing
  duration_ms UInt64 COMMENT 'How long the engine_newPayload call took in milliseconds' Codec(ZSTD(1)),

  -- Beacon context
  slot UInt32 COMMENT 'Slot number of the beacon block containing the payload' Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime COMMENT 'The wall clock time when the slot started' Codec(DoubleDelta, ZSTD(1)),
  epoch UInt32 COMMENT 'Epoch number derived from the slot' Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime COMMENT 'The wall clock time when the epoch started' Codec(DoubleDelta, ZSTD(1)),
  block_root FixedString(66) COMMENT 'Root of the beacon block (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  parent_block_root FixedString(66) COMMENT 'Root of the parent beacon block (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  proposer_index UInt32 COMMENT 'Validator index of the block proposer' Codec(ZSTD(1)),

  -- Execution payload details
  block_number UInt64 COMMENT 'Execution block number' Codec(DoubleDelta, ZSTD(1)),
  block_hash FixedString(66) COMMENT 'Execution block hash (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  parent_hash FixedString(66) COMMENT 'Parent execution block hash (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  gas_used UInt64 COMMENT 'Total gas used by all transactions in the block' Codec(ZSTD(1)),
  gas_limit UInt64 COMMENT 'Gas limit of the block' Codec(ZSTD(1)),
  tx_count UInt32 COMMENT 'Number of transactions in the block' Codec(ZSTD(1)),
  blob_count UInt32 COMMENT 'Number of blobs in the block' Codec(ZSTD(1)),

  -- Response from EL
  status LowCardinality(String) COMMENT 'Payload status returned by EL (VALID, INVALID, SYNCING, ACCEPTED, INVALID_BLOCK_HASH)',
  latest_valid_hash Nullable(FixedString(66)) COMMENT 'Latest valid hash when status is INVALID (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  validation_error Nullable(String) COMMENT 'Error message when validation fails' Codec(ZSTD(1)),

  -- Meta
  method_version LowCardinality(String) COMMENT 'Version of the engine_newPayload method (e.g., V3, V4)',

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
  meta_network_name LowCardinality(String) COMMENT 'Ethereum network name',
  meta_consensus_version LowCardinality(String) COMMENT 'Consensus client version that generated the event',
  meta_consensus_version_major LowCardinality(String) COMMENT 'Consensus client major version that generated the event',
  meta_consensus_version_minor LowCardinality(String) COMMENT 'Consensus client minor version that generated the event',
  meta_consensus_version_patch LowCardinality(String) COMMENT 'Consensus client patch version that generated the event',
  meta_consensus_implementation LowCardinality(String) COMMENT 'Consensus client implementation that generated the event',
  meta_labels Map(String, String) COMMENT 'Labels associated with the event' Codec(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, block_hash, event_date_time) COMMENT 'Contains timing and instrumentation data for engine_newPayload calls between the consensus and execution layer.';

CREATE TABLE consensus_engine_api_new_payload ON CLUSTER '{cluster}' AS consensus_engine_api_new_payload_local
ENGINE = Distributed(
  '{cluster}',
  default,
  consensus_engine_api_new_payload_local,
  cityHash64(
    slot_start_date_time,
    meta_network_name,
    meta_client_name,
    block_hash,
    event_date_time
  )
);
