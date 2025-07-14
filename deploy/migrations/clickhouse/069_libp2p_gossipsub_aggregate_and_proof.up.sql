CREATE TABLE libp2p_gossipsub_aggregate_and_proof_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    -- ensure the first time this aggregate was seen by a peer is in this table
    -- 4294967295 = UInt32 max
    version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff COMMENT 'Version of this row, to help with de-duplication we want the latest updated_date_time but lowest propagation_slot_start_diff time' CODEC(DoubleDelta, ZSTD(1)),
    event_date_time DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    slot UInt32 COMMENT 'Slot number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    epoch UInt32 COMMENT 'Epoch number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot UInt32 COMMENT 'Slot number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_slot_start_date_time DateTime COMMENT 'Start date and time of the wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_epoch UInt32 COMMENT 'Epoch number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    wallclock_epoch_start_date_time DateTime COMMENT 'Start date and time of the wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    propagation_slot_start_diff UInt32 COMMENT 'Difference in slot start time for propagation' CODEC(ZSTD(1)),
    peer_id_unique_key Int64 COMMENT 'Unique key associated with the identifier of the peer',
    message_id String COMMENT 'Identifier of the message' CODEC(ZSTD(1)),
    message_size UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    topic_layer LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    topic_fork_digest_value LowCardinality(String) COMMENT 'Fork digest value of the topic',
    topic_name LowCardinality(String) COMMENT 'Name of the topic',
    topic_encoding LowCardinality(String) COMMENT 'Encoding used for the topic',
    -- Aggregator specific fields
    aggregator_index UInt32 COMMENT 'Index of the validator who created this aggregate' CODEC(DoubleDelta, ZSTD(1)),
    -- Embedded attestation fields
    committee_index LowCardinality(String) COMMENT 'Committee index from the attestation',
    aggregation_bits String COMMENT 'Bitfield of aggregated attestation' CODEC(ZSTD(1)),
    beacon_block_root FixedString(66) COMMENT 'Root of the beacon block being attested to' CODEC(ZSTD(1)),
    source_epoch UInt32 COMMENT 'Source epoch from the attestation' CODEC(DoubleDelta, ZSTD(1)),
    source_root FixedString(66) COMMENT 'Source root from the attestation' CODEC(ZSTD(1)),
    target_epoch UInt32 COMMENT 'Target epoch from the attestation' CODEC(DoubleDelta, ZSTD(1)),
    target_root FixedString(66) COMMENT 'Target root from the attestation' CODEC(ZSTD(1)),
    -- Standard metadata fields
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
    meta_network_id Int32 COMMENT 'Network ID associated with the client' CODEC(DoubleDelta, ZSTD(1)),
    meta_network_name LowCardinality(String) COMMENT 'Name of the network associated with the client'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}/{shard}', 
    '{replica}',
    version
)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (
    slot_start_date_time,
    meta_network_name,
    meta_client_name,
    peer_id_unique_key,
    message_id
) COMMENT 'Table for libp2p gossipsub aggregate and proof data.';

-- Create distributed table
CREATE TABLE libp2p_gossipsub_aggregate_and_proof ON CLUSTER '{cluster}' AS libp2p_gossipsub_aggregate_and_proof_local
ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_gossipsub_aggregate_and_proof_local,
    cityHash64(
        slot_start_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        message_id
    )
);