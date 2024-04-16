-- Creating local and distributed tables for libp2p_publish_message
CREATE TABLE libp2p_publish_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_publish_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the messages published by the libp2p client',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN message_id 'Identifier of the message',
COMMENT COLUMN topic 'Topic associated with the message',
COMMENT COLUMN peer_id 'Identifier of the peer',
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

CREATE TABLE libp2p_publish_message on cluster '{cluster}' AS libp2p_publish_message_local
ENGINE = Distributed('{cluster}', default, libp2p_publish_message_local, rand());

-- Creating local and distributed tables for libp2p_reject_message
CREATE TABLE libp2p_reject_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    reason String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    seq_no String CODEC(ZSTD(1)),
    message_size UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_reject_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the messages rejected by the libp2p client',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN message_id 'Identifier of the message',
COMMENT COLUMN reason 'Reason for message rejection',
COMMENT COLUMN topic 'Topic associated with the message',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN seq_no 'Sequence number of the message',
COMMENT COLUMN message_size 'Size of the message in bytes',
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

CREATE TABLE libp2p_reject_message on cluster '{cluster}' AS libp2p_reject_message_local
ENGINE = Distributed('{cluster}', default, libp2p_reject_message_local, rand());

-- Creating local and distributed tables for libp2p_duplicate_message
CREATE TABLE libp2p_duplicate_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    seq_no String CODEC(ZSTD(1)),
    message_size UInt32 CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_duplicate_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the duplicate messages sent by the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN message_id 'Identifier of the message',
COMMENT COLUMN topic 'Topic associated with the message',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN seq_no 'Sequence number of the message',
COMMENT COLUMN message_size 'Size of the message in bytes',
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

CREATE TABLE libp2p_duplicate_message on cluster '{cluster}' AS libp2p_duplicate_message_local
ENGINE = Distributed('{cluster}', default, libp2p_duplicate_message_local, rand());

-- Creating local and distributed tables for libp2p_deliver_message
CREATE TABLE libp2p_deliver_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    seq_no String CODEC(ZSTD(1)),
    message_size UInt32 CODEC(ZSTD(1)),
    local Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_deliver_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the messages delivered to the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN message_id 'Identifier of the message',
COMMENT COLUMN topic 'Topic associated with the message',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN seq_no 'Sequence number of the message',
COMMENT COLUMN message_size 'Size of the message in bytes',
COMMENT COLUMN local 'Whether the message is local',
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

CREATE TABLE libp2p_deliver_message on cluster '{cluster}' AS libp2p_deliver_message_local
ENGINE = Distributed('{cluster}', default, libp2p_deliver_message_local, rand());

-- Creating local and distributed tables for libp2p_add_peer
CREATE TABLE libp2p_add_peer_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    protocol String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_add_peer_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers added to the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN protocol 'Protocol used by the peer',
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

CREATE TABLE libp2p_add_peer on cluster '{cluster}' AS  libp2p_add_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_add_peer_local, rand());

-- Creating local and distributed tables for libp2p_remove_peer
CREATE TABLE libp2p_remove_peer_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_remove_peer_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers removed from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Identifier of the peer',
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

CREATE TABLE libp2p_remove_peer on cluster '{cluster}' AS libp2p_remove_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_remove_peer_local, rand());

-- Creating tables for RPC meta data with ReplicatedReplacingMergeTree and Distributed engines
CREATE TABLE libp2p_rpc_meta_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    message_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
ORDER BY (event_date_time, control_index);

ALTER TABLE libp2p_rpc_meta_message_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC meta messages from the peer',
COMMENT COLUMN unique_key 'Unique identifier for each RPC message record',
COMMENT COLUMN control_index 'Position in the RPC meta message array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the RPC metadata',
COMMENT COLUMN message_id 'Identifier of the message',
COMMENT COLUMN topic 'Topic associated with the message',
COMMENT COLUMN updated_date_time 'Timestamp when the RPC message record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the RPC event',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the RPC',
COMMENT COLUMN meta_client_name 'Name of the client involved in the RPC',
COMMENT COLUMN meta_network_id 'Network ID associated with the RPC',
COMMENT COLUMN meta_network_name 'Network name associated with the RPC';

CREATE TABLE libp2p_rpc_meta_message AS libp2p_rpc_meta_message_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_message_local, rand());

CREATE TABLE libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    subscribe Bool,
    topic_id String CODEC(ZSTD(1)),
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
ORDER BY (event_date_time, control_index);

ALTER TABLE libp2p_rpc_meta_subscription_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC subscriptions from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each RPC subscription record',
COMMENT COLUMN control_index 'Position in the RPC meta subscription array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the RPC subscription metadata',
COMMENT COLUMN subscribe 'Boolean indicating if it is a subscription or unsubscription',
COMMENT COLUMN topic_id 'Topic ID associated with the subscription',
COMMENT COLUMN updated_date_time 'Timestamp when the RPC subscription record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the RPC subscription event',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the subscription',
COMMENT COLUMN meta_client_name 'Name of the client involved in the subscription',
COMMENT COLUMN meta_network_id 'Network ID associated with the subscription',
COMMENT COLUMN meta_network_name 'Network name associated with the subscription';

CREATE TABLE libp2p_rpc_meta_subscription AS libp2p_rpc_meta_subscription_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_subscription_local, rand());

CREATE TABLE libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    rpc_meta_unique_key Int64,
    message_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    topic_id String CODEC(ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
ORDER BY (event_date_time, control_index, message_index);

ALTER TABLE libp2p_rpc_meta_control_ihave_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I have" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "I have" control record',
COMMENT COLUMN control_index 'Position in the RPC meta control IWANT array',
COMMENT COLUMN message_index 'Position in the RPC meta control IWANT message_ids array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "I have" control metadata',
COMMENT COLUMN topic_id 'Topic ID associated with the "I have" control',
COMMENT COLUMN message_id 'Identifier of the message associated with the "I have" control',
COMMENT COLUMN updated_date_time 'Timestamp when the "I have" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "I have" control event',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the "I have" control',
COMMENT COLUMN meta_client_name 'Name of the client involved in the "I have" control',
COMMENT COLUMN meta_network_id 'Network ID associated with the "I have" control',
COMMENT COLUMN meta_network_name 'Network name associated with the "I have" control';

CREATE TABLE libp2p_rpc_meta_control_ihave AS libp2p_rpc_meta_control_ihave_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_ihave_local, rand());

CREATE TABLE libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    message_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    message_id String CODEC(ZSTD(1)),
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
ORDER BY (event_date_time, control_index, message_index);

ALTER TABLE libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I want" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "I want" control record',
COMMENT COLUMN message_index 'Position in the RPC meta control IWANT message_ids array',
COMMENT COLUMN control_index 'Position in the RPC meta control IWANT array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "I want" control metadata',
COMMENT COLUMN message_id 'Identifier of the message associated with the "I want" control',
COMMENT COLUMN updated_date_time 'Timestamp when the "I want" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "I want" control event',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the "I want" control',
COMMENT COLUMN meta_client_name 'Name of the client involved in the "I want" control',
COMMENT COLUMN meta_network_id 'Network ID associated with the "I want" control',
COMMENT COLUMN meta_network_name 'Network name associated with the "I want" control';

CREATE TABLE libp2p_rpc_meta_control_iwant AS libp2p_rpc_meta_control_iwant_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_iwant_local, rand());

CREATE TABLE libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    topic_id String CODEC(ZSTD(1)),
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
ORDER BY (event_date_time, control_index);

ALTER TABLE libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Graft" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "Graft" control record',
COMMENT COLUMN control_index 'Position in the RPC meta control GRAFT array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "Graft" control metadata',
COMMENT COLUMN topic_id 'Topic ID associated with the "Graft" control',
COMMENT COLUMN updated_date_time 'Timestamp when the "Graft" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "Graft" control event',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the "Graft" control',
COMMENT COLUMN meta_client_name 'Name of the client involved in the "Graft" control',
COMMENT COLUMN meta_network_id 'Network ID associated with the "Graft" control',
COMMENT COLUMN meta_network_name 'Network name associated with the "Graft" control';

CREATE TABLE libp2p_rpc_meta_control_graft AS libp2p_rpc_meta_control_graft_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_graft_local, rand());

CREATE TABLE libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    rpc_meta_unique_key Int64,
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    peer_id_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    topic_id String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', unique_key)
ORDER BY (event_date_time, control_index);

ALTER TABLE libp2p_rpc_meta_control_prune_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Prune" control messages from the peer.',
COMMENT COLUMN control_index 'Position in the RPC meta control PRUNE array',
COMMENT COLUMN unique_key 'Unique identifier for each "Prune" control record',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "Prune" control metadata',
COMMENT COLUMN topic_id 'Topic ID associated with the "Prune" control',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the "Prune" control',
COMMENT COLUMN updated_date_time 'Timestamp when the "Prune" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "Prune" control event',
COMMENT COLUMN peer_id 'Identifier of the peer involved in the "Prune" control',
COMMENT COLUMN meta_client_name 'Name of the client involved in the "Prune" control',
COMMENT COLUMN meta_network_id 'Network ID associated with the "Prune" control',
COMMENT COLUMN meta_network_name 'Network name associated with the "Prune" control';

CREATE TABLE libp2p_rpc_meta_control_prune ON cluster '{cluster}'  AS libp2p_rpc_meta_control_prune_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_prune_local, rand());

-- Creating local and distributed tables for libp2p_recv_rpc
CREATE TABLE libp2p_recv_rpc_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
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
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

CREATE TABLE libp2p_recv_rpc on cluster '{cluster}' AS libp2p_recv_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_recv_rpc_local, rand());

ALTER TABLE libp2p_recv_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages received by the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Peer ID of the sender',
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

-- Creating local and distributed tables for libp2p_send_rpc
CREATE TABLE libp2p_send_rpc_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_send_rpc_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages sent by the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Peer ID of the receiver',
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
CREATE TABLE libp2p_send_rpc AS libp2p_send_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_send_rpc_local, rand());

-- Creating local and distributed tables for libp2p_drop_rpc
CREATE TABLE libp2p_drop_rpc_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    send_to String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_drop_rpc_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages dropped by the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN send_to 'Identifier of the receiver',
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

CREATE TABLE libp2p_drop_rpc AS libp2p_drop_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_drop_rpc_local, rand());

-- Creating local and distributed tables for libp2p_join
CREATE TABLE libp2p_join_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_join_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the JOIN events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN topic 'Topic involved in the join event',
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

CREATE TABLE libp2p_join on cluster '{cluster}' AS libp2p_join_local
ENGINE = Distributed('{cluster}', default, libp2p_join_local, rand());

-- Creating local and distributed tables for libp2p_leave
CREATE TABLE libp2p_leave_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_leave_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the LEAVE events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN topic 'Topic involved in the leave event',
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

CREATE TABLE libp2p_leave on cluster '{cluster}' AS libp2p_leave_local
ENGINE = Distributed('{cluster}', default, libp2p_leave_local, rand());

-- Creating local and distributed tables for libp2p_graft
CREATE TABLE libp2p_graft_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_graft_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the GRAFT events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN topic 'Topic involved in the graft event',
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

CREATE TABLE libp2p_graft on cluster '{cluster}' AS libp2p_graft_local
ENGINE = Distributed('{cluster}', default, libp2p_graft_local, rand());

-- Creating local and distributed tables for libp2p_prune
CREATE TABLE libp2p_prune_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_prune_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the PRUNE events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN topic 'Topic involved in the prune event',
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

CREATE TABLE libp2p_prune on cluster '{cluster}' AS libp2p_prune_local
ENGINE = Distributed('{cluster}', default, libp2p_prune_local, rand());


-- Creating local and distributed tables for libp2p_validate_message
CREATE TABLE libp2p_validate_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    local Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_validate_message_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the VALIDATE MESSAGE events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN message_id 'ID of the message',
COMMENT COLUMN topic 'Topic involved in the validate message event',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN local 'Whether the message was local',
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

CREATE TABLE libp2p_validate_message on cluster '{cluster}' AS libp2p_validate_message_local
ENGINE = Distributed('{cluster}', default, libp2p_validate_message_local, rand());

-- Creating local and distributed tables for libp2p_throttle_peer
CREATE TABLE libp2p_throttle_peer_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_throttle_peer_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the THROTTLE PEER events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN peer_id 'Identifier of the peer',
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

CREATE TABLE libp2p_throttle_peer on cluster '{cluster}' AS libp2p_throttle_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_throttle_peer_local, rand());

-- Creating local and distributed tables for libp2p_undeliverable_message
CREATE TABLE libp2p_undeliverable_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    message_id String CODEC(ZSTD(1)),
    topic String CODEC(ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    local Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_undeliverable_message_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the UNDELIVERABLE MESSAGE events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN message_id 'ID of the message',
COMMENT COLUMN topic 'Topic involved in the undeliverable message event',
COMMENT COLUMN peer_id 'Identifier of the peer',
COMMENT COLUMN local 'Whether the message was local',
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

CREATE TABLE libp2p_undeliverable_message on cluster '{cluster}' AS libp2p_undeliverable_message_local
ENGINE = Distributed('{cluster}', default, libp2p_undeliverable_message_local, rand());

-- Creating local and distributed tables for libp2p_connected
CREATE TABLE libp2p_connected_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    remote_peer String CODEC(ZSTD(1)),
    remote_maddrs String CODEC(ZSTD(1)),
    agent_version String CODEC(ZSTD(1)),
    direction String CODEC(ZSTD(1)),
    opened DateTime CODEC(DoubleDelta, ZSTD(1)),
    transient Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_connected_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the CONNECTED events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN remote_peer 'Remote peer identifier',
COMMENT COLUMN remote_maddrs 'Multiaddresses of the remote peer',
COMMENT COLUMN agent_version 'Agent version of the remote peer',
COMMENT COLUMN direction 'Connection direction',
COMMENT COLUMN opened 'Timestamp when the connection was opened',
COMMENT COLUMN transient 'Whether the connection is transient',
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
CREATE TABLE libp2p_connected on cluster '{cluster}' AS libp2p_connected_local
ENGINE = Distributed('{cluster}', default, libp2p_connected_local, rand());

-- Creating local and distributed tables for libp2p_disconnected
CREATE TABLE libp2p_disconnected_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    session_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    remote_peer String CODEC(ZSTD(1)),
    remote_maddrs String CODEC(ZSTD(1)),
    agent_version String CODEC(ZSTD(1)),
    direction String CODEC(ZSTD(1)),
    opened DateTime CODEC(DoubleDelta, ZSTD(1)),
    transient Bool,
    meta_client_name LowCardinality(String),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String),
    meta_client_implementation LowCardinality(String),
    meta_client_os LowCardinality(String),
    meta_client_ip Nullable(IPv6) CODEC(ZSTD(1)),
    meta_client_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    meta_client_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    meta_client_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    meta_client_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(event_date_time)
ORDER BY (event_date_time, unique_key);

ALTER TABLE libp2p_disconnected_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DISCONNECTED events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN session_start_date_time 'Timestamp of the session start',
COMMENT COLUMN remote_peer 'Remote peer identifier',
COMMENT COLUMN remote_maddrs 'Multiaddresses of the remote peer',
COMMENT COLUMN agent_version 'Agent version of the remote peer',
COMMENT COLUMN direction 'Connection direction',
COMMENT COLUMN opened 'Timestamp when the connection was opened',
COMMENT COLUMN transient 'Whether the connection is transient',
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

CREATE TABLE libp2p_disconnected on cluster '{cluster}' AS libp2p_disconnected_local
ENGINE = Distributed('{cluster}', default, libp2p_disconnected_local, rand());



