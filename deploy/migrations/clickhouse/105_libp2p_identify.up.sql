CREATE TABLE IF NOT EXISTS libp2p_identify_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `remote_peer_id_unique_key` Int64 CODEC(ZSTD(1)),
    `success` Bool CODEC(ZSTD(1)),
    `error` Nullable(String) CODEC(ZSTD(1)),
    `remote_protocol` LowCardinality(String),
    `remote_transport_protocol` LowCardinality(String),
    `remote_port` UInt16 CODEC(ZSTD(1)),
    `remote_ip` Nullable(IPv6) CODEC(ZSTD(1)),
    `remote_geo_city` LowCardinality(String) CODEC(ZSTD(1)),
    `remote_geo_country` LowCardinality(String) CODEC(ZSTD(1)),
    `remote_geo_country_code` LowCardinality(String) CODEC(ZSTD(1)),
    `remote_geo_continent_code` LowCardinality(String) CODEC(ZSTD(1)),
    `remote_geo_longitude` Nullable(Float64) CODEC(ZSTD(1)),
    `remote_geo_latitude` Nullable(Float64) CODEC(ZSTD(1)),
    `remote_geo_autonomous_system_number` Nullable(UInt32) CODEC(ZSTD(1)),
    `remote_geo_autonomous_system_organization` Nullable(String) CODEC(ZSTD(1)),
    `remote_agent_implementation` LowCardinality(String),
    `remote_agent_version` LowCardinality(String),
    `remote_agent_version_major` LowCardinality(String),
    `remote_agent_version_minor` LowCardinality(String),
    `remote_agent_version_patch` LowCardinality(String),
    `remote_agent_platform` LowCardinality(String),
    `protocol_version` LowCardinality(String),
    `protocols` Array(String) CODEC(ZSTD(1)),
    `listen_addrs` Array(String) CODEC(ZSTD(1)),
    `observed_addr` String CODEC(ZSTD(1)),
    `transport` LowCardinality(String),
    `security` LowCardinality(String),
    `muxer` LowCardinality(String),
    `direction` LowCardinality(String),
    `remote_multiaddr` String CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String),
    `meta_client_id` String CODEC(ZSTD(1)),
    `meta_client_version` LowCardinality(String),
    `meta_client_implementation` LowCardinality(String),
    `meta_client_os` LowCardinality(String),
    `meta_client_ip` Nullable(IPv6) CODEC(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) CODEC(ZSTD(1)),
    `meta_client_geo_country` LowCardinality(String) CODEC(ZSTD(1)),
    `meta_client_geo_country_code` LowCardinality(String) CODEC(ZSTD(1)),
    `meta_client_geo_continent_code` LowCardinality(String) CODEC(ZSTD(1)),
    `meta_client_geo_longitude` Nullable(Float64) CODEC(ZSTD(1)),
    `meta_client_geo_latitude` Nullable(Float64) CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) CODEC(ZSTD(1)),
    `meta_network_id` Int32 CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String)
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/libp2p_identify_local/{shard}',
    '{replica}',
    updated_date_time
)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, meta_network_name, meta_client_name, remote_peer_id_unique_key, direction)
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS libp2p_identify ON CLUSTER '{cluster}' AS libp2p_identify_local
ENGINE = Distributed(
    '{cluster}',
    'default',
    'libp2p_identify_local',
    cityHash64(event_date_time, meta_network_name, meta_client_name, remote_peer_id_unique_key, direction)
);
