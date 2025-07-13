CREATE TABLE default.node_record_consensus_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the discovery module found the node' CODEC(DoubleDelta, ZSTD(1)),
    `enr` String COMMENT 'Ethereum Node Record as text' CODEC(ZSTD(1)),
    `node_id` Nullable(String) COMMENT 'ID of the Ethereum Node Record' CODEC(ZSTD(1)),
    `peer_id_unique_key` Nullable(Int64) COMMENT 'Unique key associated with the identifier of the peer',
    `timestamp` Int64 COMMENT 'Event timestamp in unix time' CODEC(DoubleDelta, ZSTD(1)),
    `name` String COMMENT 'Consensus client name' CODEC(ZSTD(1)),
    `fork_digest` String COMMENT 'Fork digest value' CODEC(ZSTD(1)),
    `next_fork_digest` Nullable(String) COMMENT 'Next fork digest of the next scheduled fork' CODEC(ZSTD(1)),
    `finalized_root` String COMMENT 'Finalized beacon block root' CODEC(ZSTD(1)),
    `finalized_epoch` UInt64 COMMENT 'Finalized epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `head_root` String COMMENT 'Head beacon block root' CODEC(ZSTD(1)),
    `head_slot` UInt64 COMMENT 'Head slot number' CODEC(DoubleDelta, ZSTD(1)),
    `cgc` Nullable(String) COMMENT 'Represents the nodes custody group count' CODEC(ZSTD(1)),
    `finalized_epoch_start_date_time` Nullable(DateTime) COMMENT 'Finalized epoch start time' CODEC(DoubleDelta, ZSTD(1)),
    `head_slot_start_date_time` Nullable(DateTime) COMMENT 'Head slot start time' CODEC(DoubleDelta, ZSTD(1)),
    `ip` Nullable(IPv6) COMMENT 'IP address of the consensus node' CODEC(ZSTD(1)),
    `geo_city` LowCardinality(String) COMMENT 'City of the consensus node' CODEC(ZSTD(1)),
    `geo_country` LowCardinality(String) COMMENT 'Country of the consensus node' CODEC(ZSTD(1)),
    `geo_country_code` LowCardinality(String) COMMENT 'Country code of the consensus node' CODEC(ZSTD(1)),
    `geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the consensus node' CODEC(ZSTD(1)),
    `geo_longitude` Nullable(Float64) COMMENT 'Longitude of the consensus node' CODEC(ZSTD(1)),
    `geo_latitude` Nullable(Float64) COMMENT 'Latitude of the consensus node' CODEC(ZSTD(1)),
    `geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the consensus node' CODEC(ZSTD(1)),
    `geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the consensus node' CODEC(ZSTD(1)),
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
        enr,
        meta_client_name
    ) COMMENT 'Contains consensus node records discovered by the Xatu discovery module.';

CREATE TABLE default.node_record_consensus ON CLUSTER '{cluster}' AS default.node_record_consensus_local ENGINE = Distributed(
    '{cluster}',
    default,
    node_record_consensus_local,
    cityHash64(
        event_date_time,
        meta_network_name,
        enr,
        meta_client_name
    )
);
