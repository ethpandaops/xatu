CREATE TABLE IF NOT EXISTS libp2p_gossipsub_inclusion_list_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot of the inclusion list' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start time of the wall clock slot' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start time of the wall clock epoch' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Propagation delay from slot start in ms' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'Index of the inclusion list committee member that signed the list' CODEC(ZSTD(1)),
    `dependent_root` FixedString(66) COMMENT 'Dependent root the inclusion list committee was computed from' CODEC(ZSTD(1)),
    `transactions_count` UInt32 COMMENT 'Number of transactions in the inclusion list' CODEC(ZSTD(1)),
    `transactions_total_bytes` UInt32 COMMENT 'Total size of the transactions in bytes' CODEC(ZSTD(1)),
    `transaction_hashes` Array(FixedString(66)) COMMENT 'Keccak-256 hashes of the transactions, in list order' CODEC(ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key for the peer identifier',
    `message_id` String COMMENT 'Identifier of the gossip message' CODEC(ZSTD(1)),
    `message_size` UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the gossipsub topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client' CODEC(ZSTD(1)),
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client' CODEC(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client' CODEC(ZSTD(1)),
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client' CODEC(ZSTD(1)),
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client' CODEC(ZSTD(1)),
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client' CODEC(ZSTD(1)),
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client' CODEC(ZSTD(1)),
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'ASN of the client' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'AS organization of the client' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY toYYYYMM(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id)
COMMENT 'Signed inclusion lists (EIP-7805 FOCIL) received on the inclusion_list gossipsub topic';

CREATE TABLE IF NOT EXISTS libp2p_gossipsub_inclusion_list ON CLUSTER '{cluster}'
AS libp2p_gossipsub_inclusion_list_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_gossipsub_inclusion_list_local', cityHash64(slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id))
COMMENT 'Signed inclusion lists (EIP-7805 FOCIL) received on the inclusion_list gossipsub topic';
