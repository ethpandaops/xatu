CREATE TABLE beacon_api_eth_v1_events_data_column_sidecar_local on cluster '{cluster}' (
    event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
    slot UInt32 Codec(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
    propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
    epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
    block_root FixedString(66) Codec(ZSTD(1)),
    column_index UInt64 Codec(ZSTD(1)),
    kzg_commitments Array(FixedString(98)) Codec(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_client_clock_drift UInt64 CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String),
    meta_consensus_version LowCardinality(String),
    meta_consensus_version_major LowCardinality(String),
    meta_consensus_version_minor LowCardinality(String),
    meta_consensus_version_patch LowCardinality(String),
    meta_consensus_implementation LowCardinality(String),
    meta_labels Map(String, String) CODEC(ZSTD(1))
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name);


CREATE TABLE beacon_api_eth_v1_events_data_column_sidecar on cluster '{cluster}' AS beacon_api_eth_v1_events_data_column_sidecar_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_data_column_sidecar_local, rand());