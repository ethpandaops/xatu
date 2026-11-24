CREATE TABLE IF NOT EXISTS libp2p_goodbye_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `remote_peer_id_unique_key` Int64 COMMENT 'Unique key of the remote peer that sent the goodbye',
    `code` UInt64 COMMENT 'Goodbye reason code sent by the remote peer, such as 1 (client shutdown), 2 (irrelevant network), 3 (fault/error) or 129 (too many peers)' CODEC(ZSTD(1)),
    `reason` LowCardinality(String) COMMENT 'Reason text the capturing client maps the code to. Unknown codes read as unknown goodbye value',
    `remote_agent_implementation` LowCardinality(String) COMMENT 'Implementation of the remote peer',
    `remote_agent_version` LowCardinality(String) COMMENT 'Version of the remote peer',
    `remote_agent_version_major` LowCardinality(String) COMMENT 'Major version of the remote peer',
    `remote_agent_version_minor` LowCardinality(String) COMMENT 'Minor version of the remote peer',
    `remote_agent_version_patch` LowCardinality(String) COMMENT 'Patch version of the remote peer',
    `remote_agent_platform` LowCardinality(String) COMMENT 'Platform of the remote peer',
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client that generated the event',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(event_date_time))
ORDER BY (meta_network_name, event_date_time, meta_client_name, remote_peer_id_unique_key, code)
COMMENT 'Contains Goodbye req/resp messages received from remote peers, with the reason code they gave for disconnecting';

CREATE TABLE IF NOT EXISTS libp2p_goodbye ON CLUSTER '{cluster}'
AS libp2p_goodbye_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_goodbye_local', cityHash64(event_date_time, meta_network_name, meta_client_name, remote_peer_id_unique_key, code))
COMMENT 'Contains Goodbye req/resp messages received from remote peers, with the reason code they gave for disconnecting';
