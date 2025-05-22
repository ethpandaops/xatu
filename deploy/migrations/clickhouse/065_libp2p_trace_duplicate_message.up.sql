CREATE TABLE libp2p_duplicate_message_local ON CLUSTER '{cluster}'
(
    updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    topic_layer LowCardinality(String) COMMENT 'Layer of the topic',
    topic_fork_digest_value LowCardinality(String) COMMENT 'Fork digest value of the topic',
    topic_name LowCardinality(String) COMMENT 'Name of the topic',
    topic_encoding LowCardinality(String) COMMENT 'Encoding of the topic',
    seq_number UInt64 COMMENT 'A linearly increasing number that is unique among messages originating from the given peer' CODEC(DoubleDelta, ZSTD(1)),
    local_delivery Bool COMMENT 'Indicates if the message was duplicated locally',
    peer_id_unique_key Int64 COMMENT 'Unique key for the peer that sent the duplicate message',
    message_id String COMMENT 'Identifier of the message' CODEC(ZSTD(1)),
    message_size UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    message_topic_name LowCardinality(String) COMMENT 'Name of the topic that the duplicate message was for',
    message_topic_encoding LowCardinality(String) COMMENT 'Encoding of the topic that the duplicate message was for',
    message_topic_fork_digest_value LowCardinality(String) COMMENT 'Fork digest value of the topic that the duplicate message was for',
    message_topic_layer LowCardinality(String) COMMENT 'Layer of the topic that the duplicate message was for',
    meta_client_name LowCardinality(String) COMMENT 'Name of the client that generated the event',
    meta_client_id String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String) COMMENT 'Version of the client that generated the event',
    meta_client_implementation LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    meta_client_os LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    meta_client_ip Nullable(IPv6) COMMENT 'IP address of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) COMMENT 'City of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) COMMENT 'Country of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) COMMENT 'Country code of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String) COMMENT 'Continent code of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64) COMMENT 'Longitude of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) COMMENT 'Latitude of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' CODEC(ZSTD(1)),
    meta_network_id Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        event_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        topic_fork_digest_value,
        topic_name,
        message_id,
        seq_number
    ) COMMENT 'Contains the details of the DUPLICATE_MESSAGE events from the libp2p client.';

CREATE TABLE libp2p_duplicate_message ON CLUSTER '{cluster}' AS libp2p_duplicate_message_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_duplicate_message_local,
    cityHash64(
        event_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        topic_fork_digest_value,
        topic_name,
        message_id,
        seq_number
    )
);