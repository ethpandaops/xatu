CREATE TABLE execution_state_size_local ON CLUSTER '{cluster}' (
  -- Timestamps
  updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' Codec(DoubleDelta, ZSTD(1)),
  event_date_time DateTime64(3) COMMENT 'When the state size measurement was taken' Codec(DoubleDelta, ZSTD(1)),

  -- Block information
  block_number UInt64 COMMENT 'Block number at which the state size was measured' Codec(DoubleDelta, ZSTD(1)),
  state_root FixedString(66) COMMENT 'State root hash of the execution layer at this block' Codec(ZSTD(1)),

  -- Account state size metrics
  accounts UInt64 COMMENT 'Total number of accounts in the state' Codec(ZSTD(1)),
  account_bytes UInt64 COMMENT 'Total bytes used by account data' Codec(ZSTD(1)),
  account_trienodes UInt64 COMMENT 'Number of trie nodes in the account trie' Codec(ZSTD(1)),
  account_trienode_bytes UInt64 COMMENT 'Total bytes used by account trie nodes' Codec(ZSTD(1)),

  -- Contract code size metrics
  contract_codes UInt64 COMMENT 'Total number of contract codes stored' Codec(ZSTD(1)),
  contract_code_bytes UInt64 COMMENT 'Total bytes used by contract code' Codec(ZSTD(1)),

  -- Storage size metrics
  storages UInt64 COMMENT 'Total number of storage slots in the state' Codec(ZSTD(1)),
  storage_bytes UInt64 COMMENT 'Total bytes used by storage data' Codec(ZSTD(1)),
  storage_trienodes UInt64 COMMENT 'Number of trie nodes in the storage trie' Codec(ZSTD(1)),
  storage_trienode_bytes UInt64 COMMENT 'Total bytes used by storage trie nodes' Codec(ZSTD(1)),

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
  meta_execution_version LowCardinality(String) COMMENT 'Execution client version that generated the event',
  meta_execution_version_major LowCardinality(String) COMMENT 'Execution client major version that generated the event',
  meta_execution_version_minor LowCardinality(String) COMMENT 'Execution client minor version that generated the event',
  meta_execution_version_patch LowCardinality(String) COMMENT 'Execution client patch version that generated the event',
  meta_execution_implementation LowCardinality(String) COMMENT 'Execution client implementation that generated the event',
  meta_labels Map(String, String) COMMENT 'Labels associated with the event' Codec(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY intDiv(block_number, 5000000)
ORDER BY (block_number, meta_network_name, meta_client_name, state_root, event_date_time) COMMENT 'Contains execution layer state size metrics including account, contract code, and storage data measurements at specific block heights.';

CREATE TABLE execution_state_size ON CLUSTER '{cluster}' AS execution_state_size_local
ENGINE = Distributed(
  '{cluster}',
  default,
  execution_state_size_local,
  cityHash64(
    block_number,
    meta_network_name,
    meta_client_name,
    state_root,
    event_date_time
  )
);
