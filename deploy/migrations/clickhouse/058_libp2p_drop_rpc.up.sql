-- libp2p_drop_rpc_local
-- unique_key = seahash(.event.id)
-- peer_id_unique_key = seahash(data.meta.peer_id + .meta_network_name)
--
-- Control messages (prune, iwant, idontwant, etc) are split off from this message
-- and stored in separate tables:
--   libp2p_rpc_meta_control_prune_local
--   libp2p_rpc_meta_control_iwant_local
--   libp2p_rpc_meta_control_idontwant_local
--   etc.
--
-- These control tables reference:
--   rpc_meta_unique_key      = libp2p_*_rpc_local.unique_key
--   peer_id_unique_key       = libp2p_*_rpc_local.peer_id_unique_key
CREATE TABLE libp2p_drop_rpc_local ON CLUSTER '{cluster}'
(
    unique_key Int64 COMMENT 'Unique identifier for each record',
    updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64 COMMENT 'Unique key associated with the identifier of the peer receiver',
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
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name)
COMMENT 'Contains the details of the RPC messages dropped by the peer.';

CREATE TABLE libp2p_drop_rpc ON CLUSTER '{cluster}' AS libp2p_drop_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_drop_rpc_local, unique_key);
