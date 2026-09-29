-- Gloas head_v2 support (EIP-7732).
--
-- The beacon-APIs head_v2 SSE event supersedes head on gloas networks:
--   {slot, block, state, payload_status, epoch_transition,
--    current_epoch_dependent_root, next_epoch_dependent_root,
--    execution_optimistic}
-- relative to head it drops previous_duty_dependent_root, renames
-- current_duty_dependent_root to current_epoch_dependent_root, and adds
-- payload_status + next_epoch_dependent_root. It lands in its own table
-- rather than beacon_api_eth_v1_events_head because the grain differs: the
-- topic legitimately fires twice for the same head as payload_status
-- transitions from empty to full, so payload_status is part of the sorting
-- key and a head can occupy two rows per client.

CREATE TABLE IF NOT EXISTS beacon_api_eth_v1_events_head_v2_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number in the beacon API event stream payload',
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started',
    `propagation_slot_start_diff` UInt32 COMMENT 'The difference between the event_date_time and the slot_start_date_time',
    `block` FixedString(66) COMMENT 'The beacon block root hash in the beacon API event stream payload',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload',
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started',
    `payload_status` LowCardinality(String) COMMENT 'The payload availability for the head block (empty/full) in the beacon API event stream payload',
    `epoch_transition` Bool COMMENT 'If the event is an epoch transition',
    `execution_optimistic` Bool COMMENT 'If the attached beacon node is running in execution optimistic mode',
    `current_epoch_dependent_root` FixedString(66) COMMENT 'The current epoch dependent root in the beacon API event stream payload',
    `next_epoch_dependent_root` FixedString(66) COMMENT 'The next epoch dependent root in the beacon API event stream payload',
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client that generated the event',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client that generated the event',
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client that generated the event',
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client that generated the event',
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client that generated the event',
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client that generated the event',
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client that generated the event',
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client that generated the event',
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event',
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event',
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Ethereum consensus client version that generated the event',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Ethereum consensus client major version that generated the event',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Ethereum consensus client minor version that generated the event',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Ethereum consensus client patch version that generated the event',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Ethereum consensus client implementation that generated the event'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, meta_client_name, block, current_epoch_dependent_root, payload_status)
COMMENT 'Contains beacon API eventstream "head_v2" data from each sentry client attached to a beacon node.';

CREATE TABLE IF NOT EXISTS beacon_api_eth_v1_events_head_v2 ON CLUSTER '{cluster}'
AS beacon_api_eth_v1_events_head_v2_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'beacon_api_eth_v1_events_head_v2_local', cityHash64(slot_start_date_time, meta_network_name, meta_client_name, block, current_epoch_dependent_root))
COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures head_v2 events. Each row represents a `head_v2` event from the Beacon API `/eth/v1/events?topics=head_v2` (gloas), indicating the chain''s canonical head has been updated. The event fires again for the same head when its payload_status transitions from empty to full, so a head can occupy two rows per client. Sentry adds client metadata and propagation timing. Partition: monthly by `slot_start_date_time`.';
