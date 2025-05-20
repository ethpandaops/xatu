CREATE TABLE default.node_record_execution_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the event was generated' CODEC(DoubleDelta, ZSTD(1)),
    `enr` String COMMENT 'Ethereum Node Record as text' CODEC(ZSTD(1)),
    `name` String COMMENT 'Execution client name' CODEC(ZSTD(1)),
    `capabilities` Array(String) COMMENT 'List of capabilities (e.g., eth/65,eth/66)' CODEC(ZSTD(1)),
    `protocol_version` String COMMENT 'Protocol version' CODEC(ZSTD(1)),
    `total_difficulty` String COMMENT 'Total difficulty of the chain' CODEC(ZSTD(1)),
    `head` String COMMENT 'Head block hash' CODEC(ZSTD(1)),
    `genesis` String COMMENT 'Genesis block hash' CODEC(ZSTD(1)),
    `fork_id_hash` String COMMENT 'Fork ID hash' CODEC(ZSTD(1)),
    `fork_id_next` String COMMENT 'Fork ID next block' CODEC(ZSTD(1)),
    `node_id` String COMMENT 'Node ID from ENR' CODEC(ZSTD(1)),
    `client_id` String COMMENT 'Client identifier from ENR' CODEC(ZSTD(1)),
    `ip4` Nullable(IPv4) COMMENT 'IPv4 address from ENR' CODEC(ZSTD(1)),
    `ip6` Nullable(IPv6) COMMENT 'IPv6 address from ENR' CODEC(ZSTD(1)),
    `tcp4` Nullable(UInt16) COMMENT 'TCP IPv4 port from ENR' CODEC(DoubleDelta, ZSTD(1)),
    `tcp6` Nullable(UInt16) COMMENT 'TCP IPv6 port from ENR' CODEC(DoubleDelta, ZSTD(1)),
    `udp4` Nullable(UInt16) COMMENT 'UDP IPv4 port from ENR' CODEC(DoubleDelta, ZSTD(1)),
    `udp6` Nullable(UInt16) COMMENT 'UDP IPv6 port from ENR' CODEC(DoubleDelta, ZSTD(1)),
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' CODEC(ZSTD(1)),
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
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(event_date_time)
ORDER BY
    (
        event_date_time,
        meta_network_name,
        node_id,
        meta_client_name
    ) COMMENT 'Contains execution node records discovered by the Xatu discovery module.';

CREATE TABLE default.node_record_execution ON CLUSTER '{cluster}' AS default.node_record_execution_local ENGINE = Distributed(
    '{cluster}',
    default,
    node_record_execution_local,
    cityHash64(
        event_date_time,
        meta_network_name,
        node_id,
        meta_client_name
    )
);