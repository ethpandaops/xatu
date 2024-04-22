-- Creating local and distributed tables for libp2p_handle_status
CREATE TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
    error Nullable(String) CODEC(ZSTD(1)),
    protocol LowCardinality(String),
    request_finalized_epoch Nullable(UInt32) CODEC(DoubleDelta, ZSTD(1)),
    request_finalized_root Nullable(String),
    request_fork_digest LowCardinality(String),
    request_head_root Nullable(FixedString(66)) CODEC(ZSTD(1)),
    request_head_slot Nullable(UInt32) CODEC(ZSTD(1)),
    response_finalized_epoch Nullable(UInt32) CODEC(DoubleDelta, ZSTD(1)),
    response_finalized_root Nullable(FixedString(66)) CODEC(ZSTD(1)),
    response_fork_digest LowCardinality(String),
    response_head_root Nullable(FixedString(66)) CODEC(ZSTD(1)),
    response_head_slot Nullable(UInt32) CODEC(DoubleDelta, ZSTD(1)),
    latency_milliseconds Float32 CODEC(ZSTD(1))
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the status handling events for libp2p peers.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN error 'Error message if the status handling failed',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer',
COMMENT COLUMN protocol 'The protocol of the status handling event',
COMMENT COLUMN request_finalized_epoch 'Requested finalized epoch',
COMMENT COLUMN request_finalized_root 'Requested finalized root',
COMMENT COLUMN request_fork_digest 'Requested fork digest',
COMMENT COLUMN request_head_root 'Requested head root',
COMMENT COLUMN request_head_slot 'Requested head slot',
COMMENT COLUMN response_finalized_epoch 'Response finalized epoch',
COMMENT COLUMN response_finalized_root 'Response finalized root',
COMMENT COLUMN response_fork_digest 'Response fork digest',
COMMENT COLUMN response_head_root 'Response head root',
COMMENT COLUMN response_head_slot 'Response head slot',
COMMENT COLUMN latency_milliseconds 'Latency of the status handling in milliseconds';

CREATE TABLE libp2p_handle_status ON CLUSTER '{cluster}' AS libp2p_handle_status_local
ENGINE = Distributed('{cluster}', default, libp2p_handle_status_local, rand());

CREATE TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
    error Nullable(String) CODEC(ZSTD(1)),
    protocol LowCardinality(String),
    attnets String CODEC(ZSTD(1)),
    seq_number UInt64 CODEC(DoubleDelta, ZSTD(1)),
    syncnets String CODEC(ZSTD(1)),
    latency_milliseconds Float32 CODEC(ZSTD(1)),
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
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the metadata handling events for libp2p peers.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN error 'Error message if the metadata handling failed',
COMMENT COLUMN protocol 'The protocol of the metadata handling event',
COMMENT COLUMN attnets 'Attestation subnets the peer is subscribed to',
COMMENT COLUMN seq_number 'Sequence number of the metadata',
COMMENT COLUMN syncnets 'Sync subnets the peer is subscribed to',
COMMENT COLUMN latency_milliseconds 'Latency of the metadata handling in milliseconds',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer involved in the RPC',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN meta_client_id 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.',
COMMENT COLUMN meta_client_version 'Version of the client that generated the event',
COMMENT COLUMN meta_client_implementation 'Implementation of the client that generated the event',
COMMENT COLUMN meta_client_os 'Operating system of the client that generated the event',
COMMENT COLUMN meta_client_ip 'IP address of the client that generated the event',
COMMENT COLUMN meta_client_geo_city 'City of the client that generated the event',
COMMENT COLUMN meta_client_geo_country 'Country of the client that generated the event',
COMMENT COLUMN meta_client_geo_country_code 'Country code of the client that generated the event',
COMMENT COLUMN meta_client_geo_continent_code 'Continent code of the client that generated the event',
COMMENT COLUMN meta_client_geo_longitude 'Longitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_latitude 'Latitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_number 'Autonomous system number of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_organization 'Autonomous system organization of the client that generated the event',
COMMENT COLUMN meta_network_id 'Ethereum network ID',
COMMENT COLUMN meta_network_name 'Ethereum network name';

CREATE TABLE libp2p_handle_metadata ON CLUSTER '{cluster}' AS libp2p_handle_metadata_local
ENGINE = Distributed('{cluster}', default, libp2p_handle_metadata_local, rand());
