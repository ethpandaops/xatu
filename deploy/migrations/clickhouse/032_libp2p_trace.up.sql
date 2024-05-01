-- Creating local and distributed tables for libp2p_peer
CREATE TABLE libp2p_peer_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    peer_id String CODEC(ZSTD(1)),
    meta_network_id Int32 CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String)
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
ORDER BY (unique_key);

ALTER TABLE libp2p_peer_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the original peer id of a seahashed peer_id + meta_network_name, commonly seen in other tables as the field peer_id_unique_key',
COMMENT COLUMN unique_key 'Unique identifier for each record, seahash of peer_id + meta_network_name',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN peer_id 'Peer ID',
COMMENT COLUMN meta_network_id 'Ethereum network ID',
COMMENT COLUMN meta_network_name 'Ethereum network name';

CREATE TABLE libp2p_peer ON CLUSTER '{cluster}' AS  libp2p_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_peer_local, unique_key);

-- Creating local and distributed tables for libp2p_add_peer
CREATE TABLE libp2p_add_peer_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
    protocol LowCardinality(String),
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_add_peer_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers added to the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer',
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

CREATE TABLE libp2p_add_peer ON CLUSTER '{cluster}' AS  libp2p_add_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_add_peer_local, unique_key);

-- Creating local and distributed tables for libp2p_remove_peer
CREATE TABLE libp2p_remove_peer_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_remove_peer_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers removed from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer',
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

CREATE TABLE libp2p_remove_peer ON CLUSTER '{cluster}' AS libp2p_remove_peer_local
ENGINE = Distributed('{cluster}', default, libp2p_remove_peer_local, unique_key);

-- Creating tables for RPC meta data with ReplicatedReplacingMergeTree and Distributed engines
CREATE TABLE libp2p_rpc_meta_message_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    message_id String CODEC(ZSTD(1)),
    topic_layer LowCardinality(String),
    topic_fork_digest_value LowCardinality(String),
    topic_name LowCardinality(String),
    topic_encoding LowCardinality(String),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, control_index, meta_network_name, meta_client_name);

ALTER TABLE libp2p_rpc_meta_message_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC meta messages from the peer',
COMMENT COLUMN unique_key 'Unique identifier for each RPC message record',
COMMENT COLUMN event_date_time 'Timestamp of the RPC event',
COMMENT COLUMN updated_date_time 'Timestamp when the RPC message record was last updated',
COMMENT COLUMN control_index 'Position in the RPC meta message array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the RPC metadata',
COMMENT COLUMN message_id 'Identifier of the message',
COMMENT COLUMN topic_layer 'Layer of the topic',
COMMENT COLUMN topic_fork_digest_value 'Fork digest value of the topic',
COMMENT COLUMN topic_name 'Name of the topic',
COMMENT COLUMN topic_encoding 'Encoding of the topic',
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

CREATE TABLE libp2p_rpc_meta_message ON CLUSTER '{cluster}' AS libp2p_rpc_meta_message_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_message_local, unique_key);

CREATE TABLE libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    subscribe Bool,
    topic_layer LowCardinality(String),
    topic_fork_digest_value LowCardinality(String),
    topic_name LowCardinality(String),
    topic_encoding LowCardinality(String),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, control_index, meta_network_name, meta_client_name);

ALTER TABLE libp2p_rpc_meta_subscription_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC subscriptions from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each RPC subscription record',
COMMENT COLUMN updated_date_time 'Timestamp when the RPC subscription record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the RPC subscription event',
COMMENT COLUMN control_index 'Position in the RPC meta subscription array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the RPC subscription metadata',
COMMENT COLUMN subscribe 'Boolean indicating if it is a subscription or unsubscription',
COMMENT COLUMN topic_layer 'Layer of the topic',
COMMENT COLUMN topic_fork_digest_value 'Fork digest value of the topic',
COMMENT COLUMN topic_name 'Name of the topic',
COMMENT COLUMN topic_encoding 'Encoding of the topic',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer involved in the subscription',
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

CREATE TABLE libp2p_rpc_meta_subscription ON CLUSTER '{cluster}' AS libp2p_rpc_meta_subscription_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_subscription_local, unique_key);

CREATE TABLE libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    message_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    topic_layer LowCardinality(String),
    topic_fork_digest_value LowCardinality(String),
    topic_name LowCardinality(String),
    topic_encoding LowCardinality(String),
    message_id String CODEC(ZSTD(1)),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, control_index, message_index, meta_network_name, meta_client_name);

ALTER TABLE libp2p_rpc_meta_control_ihave_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I have" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "I have" control record',
COMMENT COLUMN updated_date_time 'Timestamp when the "I have" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "I have" control event',
COMMENT COLUMN control_index 'Position in the RPC meta control IWANT array',
COMMENT COLUMN message_index 'Position in the RPC meta control IWANT message_ids array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "I have" control metadata',
COMMENT COLUMN topic_layer 'Layer of the topic',
COMMENT COLUMN topic_fork_digest_value 'Fork digest value of the topic',
COMMENT COLUMN topic_name 'Name of the topic',
COMMENT COLUMN topic_encoding 'Encoding of the topic',
COMMENT COLUMN message_id 'Identifier of the message associated with the "I have" control',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer involved in the I have control',
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

CREATE TABLE libp2p_rpc_meta_control_ihave ON CLUSTER '{cluster}' AS libp2p_rpc_meta_control_ihave_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_ihave_local, unique_key);

CREATE TABLE libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    message_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    message_id String CODEC(ZSTD(1)),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, control_index, message_index, meta_network_name, meta_client_name);

ALTER TABLE libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I want" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "I want" control record',
COMMENT COLUMN updated_date_time 'Timestamp when the "I want" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "I want" control event',
COMMENT COLUMN message_index 'Position in the RPC meta control IWANT message_ids array',
COMMENT COLUMN control_index 'Position in the RPC meta control IWANT array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "I want" control metadata',
COMMENT COLUMN message_id 'Identifier of the message associated with the "I want" control',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer involved in the I want control',
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

CREATE TABLE libp2p_rpc_meta_control_iwant ON CLUSTER '{cluster}' AS libp2p_rpc_meta_control_iwant_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_iwant_local, unique_key);

CREATE TABLE libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    topic_layer LowCardinality(String),
    topic_fork_digest_value LowCardinality(String),
    topic_name LowCardinality(String),
    topic_encoding LowCardinality(String),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, control_index, meta_network_name, meta_client_name);

ALTER TABLE libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Graft" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "Graft" control record',
COMMENT COLUMN updated_date_time 'Timestamp when the "Graft" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "Graft" control event',
COMMENT COLUMN control_index 'Position in the RPC meta control GRAFT array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "Graft" control metadata',
COMMENT COLUMN topic_layer 'Layer of the topic',
COMMENT COLUMN topic_fork_digest_value 'Fork digest value of the topic',
COMMENT COLUMN topic_name 'Name of the topic',
COMMENT COLUMN topic_encoding 'Encoding of the topic',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer involved in the Graft control',
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

CREATE TABLE libp2p_rpc_meta_control_graft ON CLUSTER '{cluster}' AS libp2p_rpc_meta_control_graft_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_graft_local, unique_key);

CREATE TABLE libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    control_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    rpc_meta_unique_key Int64,
    peer_id_index Int32 CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
    graft_peer_id_unique_key Int64,
    topic_layer LowCardinality(String),
    topic_fork_digest_value LowCardinality(String),
    topic_name LowCardinality(String),
    topic_encoding LowCardinality(String),
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
ORDER BY (event_date_time, unique_key, control_index, meta_network_name, meta_client_name);

ALTER TABLE libp2p_rpc_meta_control_prune_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Prune" control messages from the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each "Prune" control record',
COMMENT COLUMN updated_date_time 'Timestamp when the "Prune" control record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the "Prune" control event',
COMMENT COLUMN control_index 'Position in the RPC meta control PRUNE array',
COMMENT COLUMN rpc_meta_unique_key 'Unique key associated with the "Prune" control metadata',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer involved in the Prune control',
COMMENT COLUMN graft_peer_id_unique_key 'Unique key associated with the identifier of the graft peer involved in the Prune control',
COMMENT COLUMN topic_layer 'Layer of the topic',
COMMENT COLUMN topic_fork_digest_value 'Fork digest value of the topic',
COMMENT COLUMN topic_name 'Name of the topic',
COMMENT COLUMN topic_encoding 'Encoding of the topic',
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

CREATE TABLE libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}' AS libp2p_rpc_meta_control_prune_local
ENGINE = Distributed('{cluster}', default, libp2p_rpc_meta_control_prune_local, unique_key);

-- Creating local and distributed tables for libp2p_recv_rpc
CREATE TABLE libp2p_recv_rpc_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_recv_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages received by the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer sender',
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

CREATE TABLE libp2p_recv_rpc ON CLUSTER '{cluster}' AS libp2p_recv_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_recv_rpc_local, unique_key);

-- Creating local and distributed tables for libp2p_send_rpc
CREATE TABLE libp2p_send_rpc_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_send_rpc_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages sent by the peer.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer receiver',
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

CREATE TABLE libp2p_send_rpc ON CLUSTER '{cluster}' AS libp2p_send_rpc_local
ENGINE = Distributed('{cluster}', default, libp2p_send_rpc_local, unique_key);

-- Creating local and distributed tables for libp2p_join
CREATE TABLE libp2p_join_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    topic_layer LowCardinality(String),
    topic_fork_digest_value LowCardinality(String),
    topic_name LowCardinality(String),
    topic_encoding LowCardinality(String),
    peer_id_unique_key Int64,
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_join_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the JOIN events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN topic_layer 'Layer of the topic',
COMMENT COLUMN topic_fork_digest_value 'Fork digest value of the topic',
COMMENT COLUMN topic_name 'Name of the topic',
COMMENT COLUMN topic_encoding 'Encoding of the topic',
COMMENT COLUMN peer_id_unique_key 'Unique key associated with the identifier of the peer that joined the topic',
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

CREATE TABLE libp2p_join ON CLUSTER '{cluster}' AS libp2p_join_local
ENGINE = Distributed('{cluster}', default, libp2p_join_local, unique_key);

-- Creating local and distributed tables for libp2p_connected
CREATE TABLE libp2p_connected_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    remote_peer_id_unique_key Int64,
    remote_protocol LowCardinality(String),
    remote_transport_protocol LowCardinality(String),
    remote_port UInt16 CODEC(ZSTD(1)),
    remote_ip Nullable(IPv6) CODEC(ZSTD(1)),
    remote_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    remote_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    remote_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    remote_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    remote_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    remote_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    remote_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    remote_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    remote_agent_implementation LowCardinality(String),
    remote_agent_version LowCardinality(String),
    remote_agent_version_major LowCardinality(String),
    remote_agent_version_minor LowCardinality(String),
    remote_agent_version_patch LowCardinality(String),
    remote_agent_platform LowCardinality(String),
    direction LowCardinality(String),
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_connected_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the CONNECTED events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN remote_peer_id_unique_key 'Unique key associated with the identifier of the remote peer',
COMMENT COLUMN remote_protocol 'Protocol of the remote peer',
COMMENT COLUMN remote_transport_protocol 'Transport protocol of the remote peer',
COMMENT COLUMN remote_port 'Port of the remote peer',
COMMENT COLUMN remote_ip 'IP address of the remote peer that generated the event',
COMMENT COLUMN remote_geo_city 'City of the remote peer that generated the event',
COMMENT COLUMN remote_geo_country 'Country of the remote peer that generated the event',
COMMENT COLUMN remote_geo_country_code 'Country code of the remote peer that generated the event',
COMMENT COLUMN remote_geo_continent_code 'Continent code of the remote peer that generated the event',
COMMENT COLUMN remote_geo_longitude 'Longitude of the remote peer that generated the event',
COMMENT COLUMN remote_geo_latitude 'Latitude of the remote peer that generated the event',
COMMENT COLUMN remote_geo_autonomous_system_number 'Autonomous system number of the remote peer that generated the event',
COMMENT COLUMN remote_geo_autonomous_system_organization 'Autonomous system organization of the remote peer that generated the event',
COMMENT COLUMN remote_agent_implementation 'Implementation of the remote peer',
COMMENT COLUMN remote_agent_version 'Version of the remote peer',
COMMENT COLUMN remote_agent_version_major 'Major version of the remote peer',
COMMENT COLUMN remote_agent_version_minor 'Minor version of the remote peer',
COMMENT COLUMN remote_agent_version_patch 'Patch version of the remote peer',
COMMENT COLUMN remote_agent_platform 'Platform of the remote peer',
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

CREATE TABLE libp2p_connected ON CLUSTER '{cluster}' AS libp2p_connected_local
ENGINE = Distributed('{cluster}', default, libp2p_connected_local, unique_key);

-- Creating local and distributed tables for libp2p_disconnected
CREATE TABLE libp2p_disconnected_local ON CLUSTER '{cluster}'
(
    unique_key Int64,
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    remote_peer_id_unique_key Int64,
    remote_protocol LowCardinality(String),
    remote_transport_protocol LowCardinality(String),
    remote_port UInt16 CODEC(ZSTD(1)),
    remote_ip Nullable(IPv6) CODEC(ZSTD(1)),
    remote_geo_city LowCardinality(String) CODEC(ZSTD(1)),
    remote_geo_country LowCardinality(String) CODEC(ZSTD(1)),
    remote_geo_country_code LowCardinality(String) CODEC(ZSTD(1)),
    remote_geo_continent_code LowCardinality(String)  CODEC(ZSTD(1)),
    remote_geo_longitude Nullable(Float64)  CODEC(ZSTD(1)),
    remote_geo_latitude Nullable(Float64) CODEC(ZSTD(1)),
    remote_geo_autonomous_system_number Nullable(UInt32) CODEC(ZSTD(1)),
    remote_geo_autonomous_system_organization Nullable(String) CODEC(ZSTD(1)),
    remote_agent_implementation LowCardinality(String),
    remote_agent_version LowCardinality(String),
    remote_agent_version_major LowCardinality(String),
    remote_agent_version_minor LowCardinality(String),
    remote_agent_version_patch LowCardinality(String),
    remote_agent_platform LowCardinality(String),
    direction LowCardinality(String),
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
ORDER BY (event_date_time, unique_key, meta_network_name, meta_client_name);

ALTER TABLE libp2p_disconnected_local  ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DISCONNECTED events from the libp2p client.',
COMMENT COLUMN unique_key 'Unique identifier for each record',
COMMENT COLUMN updated_date_time 'Timestamp when the record was last updated',
COMMENT COLUMN event_date_time 'Timestamp of the event',
COMMENT COLUMN remote_peer_id_unique_key 'Unique key associated with the identifier of the remote peer',
COMMENT COLUMN remote_protocol 'Protocol of the remote peer',
COMMENT COLUMN remote_transport_protocol 'Transport protocol of the remote peer',
COMMENT COLUMN remote_port 'Port of the remote peer',
COMMENT COLUMN remote_ip 'IP address of the remote peer that generated the event',
COMMENT COLUMN remote_geo_city 'City of the remote peer that generated the event',
COMMENT COLUMN remote_geo_country 'Country of the remote peer that generated the event',
COMMENT COLUMN remote_geo_country_code 'Country code of the remote peer that generated the event',
COMMENT COLUMN remote_geo_continent_code 'Continent code of the remote peer that generated the event',
COMMENT COLUMN remote_geo_longitude 'Longitude of the remote peer that generated the event',
COMMENT COLUMN remote_geo_latitude 'Latitude of the remote peer that generated the event',
COMMENT COLUMN remote_geo_autonomous_system_number 'Autonomous system number of the remote peer that generated the event',
COMMENT COLUMN remote_geo_autonomous_system_organization 'Autonomous system organization of the remote peer that generated the event',
COMMENT COLUMN remote_agent_implementation 'Implementation of the remote peer',
COMMENT COLUMN remote_agent_version 'Version of the remote peer',
COMMENT COLUMN remote_agent_version_major 'Major version of the remote peer',
COMMENT COLUMN remote_agent_version_minor 'Minor version of the remote peer',
COMMENT COLUMN remote_agent_version_patch 'Patch version of the remote peer',
COMMENT COLUMN remote_agent_platform 'Platform of the remote peer',
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

CREATE TABLE libp2p_disconnected ON CLUSTER '{cluster}' AS libp2p_disconnected_local
ENGINE = Distributed('{cluster}', default, libp2p_disconnected_local, unique_key);
