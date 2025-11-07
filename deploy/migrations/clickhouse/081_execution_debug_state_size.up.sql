CREATE TABLE execution_debug_state_size_local ON CLUSTER '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  account_bytes UInt64 Codec(ZSTD(1)),
  account_trienode_bytes UInt64 Codec(ZSTD(1)),
  account_trienodes UInt64 Codec(ZSTD(1)),
  accounts UInt64 Codec(ZSTD(1)),
  block_number UInt64 Codec(DoubleDelta, ZSTD(1)),
  contract_code_bytes UInt64 Codec(ZSTD(1)),
  contract_codes UInt64 Codec(ZSTD(1)),
  state_root FixedString(66) Codec(ZSTD(1)),
  storage_bytes UInt64 Codec(ZSTD(1)),
  storage_trienode_bytes UInt64 Codec(ZSTD(1)),
  storage_trienodes UInt64 Codec(ZSTD(1)),
  storages UInt64 Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_execution_version LowCardinality(String),
  meta_execution_version_major LowCardinality(String),
  meta_execution_version_minor LowCardinality(String),
  meta_execution_version_patch LowCardinality(String),
  meta_execution_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1))
) ENGINE = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(event_date_time)
ORDER BY (event_date_time, meta_network_name, meta_client_name, block_number);

CREATE TABLE execution_debug_state_size ON CLUSTER '{cluster}' AS execution_debug_state_size_local
ENGINE = Distributed('{cluster}', default, execution_debug_state_size_local, rand());
