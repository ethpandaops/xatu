CREATE TABLE beacon_api_eth_v1_beacon_committee_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  committee_index LowCardinality(String),
  validators Array(UInt32) Codec(ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
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
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1))
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_beacon_committee on cluster '{cluster}' AS beacon_api_eth_v1_beacon_committee_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_beacon_committee_local, rand());
