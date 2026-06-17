-- Reverts 004: renames local_peer_id_unique_key -> peer_id_unique_key on the
-- libp2p_join and libp2p_leave tables, preserving existing rows by staging them
-- in a temporary backup and copying them back.

-- 1. Stage existing libp2p_join rows into a temporary backup (new schema).
CREATE TABLE IF NOT EXISTS libp2p_join_migrate_bak_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding of the topic',
    `local_peer_id_unique_key` Int64 COMMENT 'Unique key derived from the libp2p peer.ID of the local host that joined the topic',
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
ORDER BY (meta_network_name, event_date_time, meta_client_name, local_peer_id_unique_key, topic_fork_digest_value, topic_name);

CREATE TABLE IF NOT EXISTS libp2p_join_migrate_bak ON CLUSTER '{cluster}'
AS libp2p_join_migrate_bak_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_join_migrate_bak_local', cityHash64(event_date_time, meta_network_name, meta_client_name, local_peer_id_unique_key, topic_fork_digest_value, topic_name));

INSERT INTO libp2p_join_migrate_bak SELECT * FROM libp2p_join SETTINGS distributed_foreground_insert = 1;

-- 1b. Stage existing libp2p_leave rows into a temporary backup (new schema).
CREATE TABLE IF NOT EXISTS libp2p_leave_migrate_bak_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding of the topic',
    `local_peer_id_unique_key` Int64 COMMENT 'Unique key derived from the libp2p peer.ID of the local host that left the topic',
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
ORDER BY (meta_network_name, event_date_time, meta_client_name, local_peer_id_unique_key, topic_fork_digest_value, topic_name);

CREATE TABLE IF NOT EXISTS libp2p_leave_migrate_bak ON CLUSTER '{cluster}'
AS libp2p_leave_migrate_bak_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_leave_migrate_bak_local', cityHash64(event_date_time, meta_network_name, meta_client_name, local_peer_id_unique_key, topic_fork_digest_value, topic_name));

INSERT INTO libp2p_leave_migrate_bak SELECT * FROM libp2p_leave SETTINGS distributed_foreground_insert = 1;

-- 2. Drop the live tables.
DROP TABLE IF EXISTS libp2p_join ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_join_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_leave ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_leave_local ON CLUSTER '{cluster}' SYNC;

-- 3. Recreate with the old schema (peer_id_unique_key), canonical ZK path.
CREATE TABLE IF NOT EXISTS libp2p_join_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding of the topic',
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer that joined the topic',
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
ORDER BY (meta_network_name, event_date_time, meta_client_name, peer_id_unique_key, topic_fork_digest_value, topic_name)
COMMENT 'Contains the details of the JOIN events from the libp2p client.';

CREATE TABLE IF NOT EXISTS libp2p_leave_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding of the topic',
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer that left the topic',
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
ORDER BY (meta_network_name, event_date_time, meta_client_name, peer_id_unique_key, topic_fork_digest_value, topic_name)
COMMENT 'Contains the details of the LEAVE events from the libp2p client.';

CREATE TABLE IF NOT EXISTS libp2p_join ON CLUSTER '{cluster}'
AS libp2p_join_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_join_local', cityHash64(event_date_time, meta_network_name, meta_client_name, peer_id_unique_key, topic_fork_digest_value, topic_name));

CREATE TABLE IF NOT EXISTS libp2p_leave ON CLUSTER '{cluster}'
AS libp2p_leave_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_leave_local', cityHash64(event_date_time, meta_network_name, meta_client_name, peer_id_unique_key, topic_fork_digest_value, topic_name));

-- 4. Restore staged rows. local_peer_id_unique_key maps back into peer_id_unique_key.
INSERT INTO libp2p_join SELECT
    updated_date_time, event_date_time, topic_layer, topic_fork_digest_value, topic_name, topic_encoding,
    local_peer_id_unique_key AS peer_id_unique_key,
    meta_client_name, meta_client_version, meta_client_implementation, meta_client_os,
    meta_client_ip, meta_client_geo_city, meta_client_geo_country, meta_client_geo_country_code,
    meta_client_geo_continent_code, meta_client_geo_longitude, meta_client_geo_latitude,
    meta_client_geo_autonomous_system_number, meta_client_geo_autonomous_system_organization, meta_network_name
FROM libp2p_join_migrate_bak SETTINGS distributed_foreground_insert = 1;

INSERT INTO libp2p_leave SELECT
    updated_date_time, event_date_time, topic_layer, topic_fork_digest_value, topic_name, topic_encoding,
    local_peer_id_unique_key AS peer_id_unique_key,
    meta_client_name, meta_client_version, meta_client_implementation, meta_client_os,
    meta_client_ip, meta_client_geo_city, meta_client_geo_country, meta_client_geo_country_code,
    meta_client_geo_continent_code, meta_client_geo_longitude, meta_client_geo_latitude,
    meta_client_geo_autonomous_system_number, meta_client_geo_autonomous_system_organization, meta_network_name
FROM libp2p_leave_migrate_bak SETTINGS distributed_foreground_insert = 1;

-- 5. Drop the temporary backups.
DROP TABLE IF EXISTS libp2p_join_migrate_bak ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_join_migrate_bak_local ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_leave_migrate_bak ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS libp2p_leave_migrate_bak_local ON CLUSTER '{cluster}' SYNC;
