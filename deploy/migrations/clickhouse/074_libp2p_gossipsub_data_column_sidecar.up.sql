CREATE TABLE libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - propagation_slot_start_diff COMMENT 'Version of this row, to help with de-duplication we want the latest updated_date_time but lowest propagation_slot_start_diff time' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number associated with the event' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference in slot start time for propagation' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'The proposer index of the beacon block' CODEC(ZSTD(1)),
    `column_index` UInt64 COMMENT 'Column index associated with the record' CODEC(ZSTD(1)),
    `parent_root` FixedString(66) COMMENT 'Parent root of the beacon block' CODEC(ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'State root of the beacon block' CODEC(ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key associated with the identifier of the peer',
    `message_id` String COMMENT 'Identifier of the message' CODEC(ZSTD(1)),
    `message_size` UInt32 COMMENT 'Size of the message in bytes' CODEC(ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
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
    `meta_network_id` Int32 COMMENT 'Network ID associated with the client' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(event_date_time)
ORDER BY
    (
        slot_start_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        message_id
    ) COMMENT 'Table for libp2p gossipsub data column sidecar data';

CREATE TABLE libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}' AS libp2p_gossipsub_data_column_sidecar_local ENGINE = Distributed(
    '{cluster}',
    default,
    libp2p_gossipsub_data_column_sidecar_local,
    cityHash64(
        slot_start_date_time,
        meta_network_name,
        meta_client_name,
        peer_id_unique_key,
        message_id
    )
);
