CREATE TABLE execution_block_metrics_local ON CLUSTER '{cluster}' (
  -- Timestamps
  updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' Codec(DoubleDelta, ZSTD(1)),
  event_date_time DateTime64(3) COMMENT 'When the event was received' Codec(DoubleDelta, ZSTD(1)),

  -- Source tracking
  source LowCardinality(String) COMMENT 'Data source (e.g., client-logs)',

  -- Block info
  block_number UInt64 COMMENT 'Execution block number' Codec(DoubleDelta, ZSTD(1)),
  block_hash FixedString(66) COMMENT 'Execution block hash (hex encoded with 0x prefix)' Codec(ZSTD(1)),
  gas_used UInt64 COMMENT 'Total gas used by all transactions in the block' Codec(ZSTD(1)),
  tx_count UInt32 COMMENT 'Number of transactions in the block' Codec(ZSTD(1)),

  -- Timing in milliseconds
  execution_ms Float64 COMMENT 'Time spent executing transactions in milliseconds' Codec(ZSTD(1)),
  state_read_ms Float64 COMMENT 'Time spent reading state in milliseconds' Codec(ZSTD(1)),
  state_hash_ms Float64 COMMENT 'Time spent computing state hash in milliseconds' Codec(ZSTD(1)),
  commit_ms Float64 COMMENT 'Time spent committing state changes in milliseconds' Codec(ZSTD(1)),
  total_ms Float64 COMMENT 'Total time for block processing in milliseconds' Codec(ZSTD(1)),

  -- Throughput
  mgas_per_sec Float64 COMMENT 'Throughput in million gas per second' Codec(ZSTD(1)),

  -- State reads
  state_reads_accounts UInt64 COMMENT 'Number of account reads' Codec(ZSTD(1)),
  state_reads_storage_slots UInt64 COMMENT 'Number of storage slot reads' Codec(ZSTD(1)),
  state_reads_code UInt64 COMMENT 'Number of code reads' Codec(ZSTD(1)),
  state_reads_code_bytes UInt64 COMMENT 'Total bytes of code read' Codec(ZSTD(1)),

  -- State writes
  state_writes_accounts UInt64 COMMENT 'Number of account writes' Codec(ZSTD(1)),
  state_writes_accounts_deleted UInt64 COMMENT 'Number of accounts deleted' Codec(ZSTD(1)),
  state_writes_storage_slots UInt64 COMMENT 'Number of storage slot writes' Codec(ZSTD(1)),
  state_writes_storage_slots_deleted UInt64 COMMENT 'Number of storage slots deleted' Codec(ZSTD(1)),
  state_writes_code UInt64 COMMENT 'Number of code writes' Codec(ZSTD(1)),
  state_writes_code_bytes UInt64 COMMENT 'Total bytes of code written' Codec(ZSTD(1)),

  -- Cache metrics
  account_cache_hits Int64 COMMENT 'Number of account cache hits' Codec(ZSTD(1)),
  account_cache_misses Int64 COMMENT 'Number of account cache misses' Codec(ZSTD(1)),
  account_cache_hit_rate Float64 COMMENT 'Account cache hit rate as percentage' Codec(ZSTD(1)),
  storage_cache_hits Int64 COMMENT 'Number of storage cache hits' Codec(ZSTD(1)),
  storage_cache_misses Int64 COMMENT 'Number of storage cache misses' Codec(ZSTD(1)),
  storage_cache_hit_rate Float64 COMMENT 'Storage cache hit rate as percentage' Codec(ZSTD(1)),
  code_cache_hits Int64 COMMENT 'Number of code cache hits' Codec(ZSTD(1)),
  code_cache_misses Int64 COMMENT 'Number of code cache misses' Codec(ZSTD(1)),
  code_cache_hit_rate Float64 COMMENT 'Code cache hit rate as percentage' Codec(ZSTD(1)),
  code_cache_hit_bytes Int64 COMMENT 'Total bytes of code cache hits' Codec(ZSTD(1)),
  code_cache_miss_bytes Int64 COMMENT 'Total bytes of code cache misses' Codec(ZSTD(1)),

  -- Standard metadata fields
  meta_client_name LowCardinality(String) COMMENT 'Name of the client that generated the event',
  meta_client_id String COMMENT 'Unique Session ID of the client that generated the event' Codec(ZSTD(1)),
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
  meta_labels Map(String, String) COMMENT 'Labels associated with the event' Codec(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
  '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
  '{replica}',
  updated_date_time
)
PARTITION BY intDiv(block_number, 5000000)
ORDER BY (block_number, meta_network_name, meta_client_name, event_date_time)
COMMENT 'Contains detailed performance metrics from execution client structured logging for block execution';

CREATE TABLE execution_block_metrics ON CLUSTER '{cluster}' AS execution_block_metrics_local
ENGINE = Distributed(
  '{cluster}',
  default,
  execution_block_metrics_local,
  cityHash64(
    block_number,
    meta_network_name,
    meta_client_name
  )
);
