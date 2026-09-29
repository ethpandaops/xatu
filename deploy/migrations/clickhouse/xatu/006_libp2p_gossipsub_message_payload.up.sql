CREATE TABLE IF NOT EXISTS libp2p_gossipsub_message_payload_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the message was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the message was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the message was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the message was received' CODEC(DoubleDelta, ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding of the topic',
    `message_id` String COMMENT 'Gossipsub message ID, derived from the message contents' CODEC(ZSTD(1)),
    `message_size` UInt32 COMMENT 'Size of the message payload in bytes' CODEC(ZSTD(1)),
    `message_data` String COMMENT 'Raw gossipsub message payload as received off the wire (snappy-framed SSZ)' CODEC(ZSTD(1)),
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
PARTITION BY (meta_network_name, toDate(wallclock_slot_start_date_time))
ORDER BY (meta_network_name, wallclock_slot_start_date_time, topic_fork_digest_value, topic_name, message_id)
COMMENT 'Contains raw gossipsub message payloads keyed by message ID. Message IDs are content-derived, and the sorting key deliberately excludes observation-specific columns (peer, client, receive time, validation outcome) so identical messages captured by multiple clients deduplicate on merge. Deduplication is best-effort: clients that receive the same message on opposite sides of a wallclock slot or partition boundary keep one row per side. Per-observation detail lives in the libp2p_deliver_message and libp2p_reject_message tables.';

CREATE TABLE IF NOT EXISTS libp2p_gossipsub_message_payload ON CLUSTER '{cluster}'
AS libp2p_gossipsub_message_payload_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_gossipsub_message_payload_local', cityHash64(meta_network_name, topic_fork_digest_value, topic_name, message_id))
COMMENT 'Contains raw gossipsub message payloads keyed by message ID. The sharding key is content-derived so duplicates from multiple capture clients land on the same shard and deduplicate.';
