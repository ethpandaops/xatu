CREATE TABLE IF NOT EXISTS libp2p_gossipsub_partial_data_column_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot of the block the partial belongs to. 0 when the capture point could not resolve it' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot. Unix epoch when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch of the block the partial belongs to. 0 when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch. Unix epoch when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the event was captured' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the event was captured' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the event was captured' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the event was captured' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Milliseconds between the slot start and the capture. 0 when the slot is unknown' CODEC(ZSTD(1)),
    `direction` LowCardinality(String) COMMENT 'RPC direction relative to the capturing node: RECV, SEND or DROP',
    `capture_point` LowCardinality(String) COMMENT 'WIRE when read from the raw RPC, APP when read after the gossipsub router filtered it. Counts are not comparable across capture points',
    `peer_id_unique_key` Int64 COMMENT 'Unique key of the remote peer: the sender for RECV, the recipient for SEND and DROP',
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
    `group_id` String COMMENT 'Hex encoded partial messages group ID: version byte followed by the SSZ encoded block identity' CODEC(ZSTD(1)),
    `group_version` UInt8 COMMENT 'Group ID version byte: 0 for fulu, 1 for gloas' CODEC(ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'Root of the beacon block the partial belongs to' CODEC(ZSTD(1)),
    `column_index` UInt64 COMMENT 'Data column index' CODEC(ZSTD(1)),
    `has_message` Bool COMMENT 'Whether the RPC part carried a partial message body',
    `message_size` UInt32 COMMENT 'Size of the SSZ encoded partial message in bytes. 0 for metadata-only parts' CODEC(ZSTD(1)),
    `cells_count` UInt32 COMMENT 'Number of cells carried in the partial message' CODEC(ZSTD(1)),
    `cells_bitmap` String COMMENT 'Hex encoded SSZ cells_present_bitmap of the partial message' CODEC(ZSTD(1)),
    `bitmap_length` UInt32 COMMENT 'Length of the cells bitlist, equal to the block blob count' CODEC(ZSTD(1)),
    `has_header` Bool COMMENT 'Whether the partial message carried the partial data column header',
    `kzg_commitments_count` UInt32 COMMENT 'Number of KZG commitments in the header. 0 when no header was carried' CODEC(ZSTD(1)),
    `proposer_index` UInt32 COMMENT 'Proposer index from the header signed block header. 0 when no header was carried' CODEC(ZSTD(1)),
    `has_metadata` Bool COMMENT 'Whether the RPC part carried parts metadata',
    `metadata_size` UInt32 COMMENT 'Size of the SSZ encoded parts metadata in bytes' CODEC(ZSTD(1)),
    `metadata_available_count` UInt32 COMMENT 'Number of set bits in the parts metadata available bitlist' CODEC(ZSTD(1)),
    `metadata_requests_count` UInt32 COMMENT 'Number of set bits in the parts metadata requests bitlist' CODEC(ZSTD(1)),
    `metadata_available` String COMMENT 'Hex encoded SSZ parts metadata available bitlist' CODEC(ZSTD(1)),
    `metadata_requests` String COMMENT 'Hex encoded SSZ parts metadata requests bitlist' CODEC(ZSTD(1)),
    `unique_key` Int64 COMMENT 'Unique key of the event, derived from the event ID',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(event_date_time))
ORDER BY (meta_network_name, wallclock_slot_start_date_time, meta_client_name, peer_id_unique_key, column_index, unique_key)
COMMENT 'Contains gossipsub partial messages extension RPC parts on data column topics, one row per observation in either direction';

CREATE TABLE IF NOT EXISTS libp2p_gossipsub_partial_data_column ON CLUSTER '{cluster}'
AS libp2p_gossipsub_partial_data_column_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_gossipsub_partial_data_column_local', cityHash64(wallclock_slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, column_index, unique_key))
COMMENT 'Contains gossipsub partial messages extension RPC parts on data column topics, one row per observation in either direction';

CREATE TABLE IF NOT EXISTS libp2p_data_column_completed_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot of the block the column belongs to. 0 when the capture point could not resolve it' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot. Unix epoch when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch of the block the column belongs to. 0 when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch. Unix epoch when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the column completed' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the column completed' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the column completed' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the column completed' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Milliseconds between the slot start and the completion. 0 when the slot is unknown' CODEC(ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key of the peer whose message completed the column. 0 when no peer did',
    `beacon_block_root` FixedString(66) COMMENT 'Root of the beacon block the column belongs to' CODEC(ZSTD(1)),
    `column_index` UInt64 COMMENT 'Data column index' CODEC(ZSTD(1)),
    `completed_via` LowCardinality(String) COMMENT 'Path that completed the column: FULL_GOSSIP, PARTIAL_GOSSIP, EL, RECONSTRUCTION or UNKNOWN',
    `cells_total` UInt32 COMMENT 'Number of cells in the completed column' CODEC(ZSTD(1)),
    `cells_from_gossip` UInt32 COMMENT 'Number of cells received over gossip' CODEC(ZSTD(1)),
    `cells_from_el` UInt32 COMMENT 'Number of cells built from blobs returned by the execution layer' CODEC(ZSTD(1)),
    `contributing_peers` UInt32 COMMENT 'Number of distinct peers that contributed cells' CODEC(ZSTD(1)),
    `kzg_commitments_count` UInt32 COMMENT 'Number of KZG commitments of the block' CODEC(ZSTD(1)),
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(event_date_time))
ORDER BY (meta_network_name, slot_start_date_time, meta_client_name, beacon_block_root, column_index)
COMMENT 'Contains custody data column completions, one row per node per column whatever the path that completed it';

CREATE TABLE IF NOT EXISTS libp2p_data_column_completed ON CLUSTER '{cluster}'
AS libp2p_data_column_completed_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_data_column_completed_local', cityHash64(slot_start_date_time, meta_network_name, meta_client_name, beacon_block_root, column_index))
COMMENT 'Contains custody data column completions, one row per node per column whatever the path that completed it';

CREATE TABLE IF NOT EXISTS libp2p_gossipsub_partial_data_column_outcome_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot of the block the partial belongs to. 0 when the capture point could not resolve it' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot. Unix epoch when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch of the block the partial belongs to. 0 when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch. Unix epoch when the slot is unknown' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Slot number of the wall clock when the verdict was reached' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start date and time of the wall clock slot when the verdict was reached' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Epoch number of the wall clock when the verdict was reached' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start date and time of the wall clock epoch when the verdict was reached' CODEC(DoubleDelta, ZSTD(1)),
    `peer_id_unique_key` Int64 COMMENT 'Unique key of the peer that sent the partial message',
    `topic_layer` LowCardinality(String) COMMENT 'Layer of the topic in the gossipsub protocol',
    `topic_fork_digest_value` LowCardinality(String) COMMENT 'Fork digest value of the topic',
    `topic_name` LowCardinality(String) COMMENT 'Name of the topic',
    `topic_encoding` LowCardinality(String) COMMENT 'Encoding used for the topic',
    `group_id` String COMMENT 'Hex encoded partial messages group ID: version byte followed by the SSZ encoded block identity' CODEC(ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'Root of the beacon block the partial belongs to' CODEC(ZSTD(1)),
    `column_index` UInt64 COMMENT 'Data column index' CODEC(ZSTD(1)),
    `stage` LowCardinality(String) COMMENT 'Validation stage that produced the verdict: DECODE, HEADER or CELLS',
    `outcome` LowCardinality(String) COMMENT 'Validation verdict: ACCEPT, IGNORE or REJECT',
    `reason` LowCardinality(String) COMMENT 'Client reported reason for the verdict. Empty when none was given',
    `cells_count` UInt32 COMMENT 'Number of cells carried in the partial message' CODEC(ZSTD(1)),
    `new_cells_count` UInt32 COMMENT 'Number of carried cells the node did not already have' CODEC(ZSTD(1)),
    `unique_key` Int64 COMMENT 'Unique key of the event, derived from the event ID',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Name of the network associated with the client'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(event_date_time))
ORDER BY (meta_network_name, wallclock_slot_start_date_time, meta_client_name, peer_id_unique_key, column_index, unique_key)
COMMENT 'Contains validation verdicts for received gossipsub partial data column messages';

CREATE TABLE IF NOT EXISTS libp2p_gossipsub_partial_data_column_outcome ON CLUSTER '{cluster}'
AS libp2p_gossipsub_partial_data_column_outcome_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'libp2p_gossipsub_partial_data_column_outcome_local', cityHash64(wallclock_slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, column_index, unique_key))
COMMENT 'Contains validation verdicts for received gossipsub partial data column messages';

-- Appended (no AFTER) so existing pkg/proto/clickhouse field numbers stay stable.
ALTER TABLE libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS requests_partial Nullable(Bool)
        COMMENT 'Whether the peer asked to receive gossipsub partial messages on the topic. Null when the tracer cannot see the partial messages subscription options' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS supports_sending_partial Nullable(Bool)
        COMMENT 'Whether the peer can send gossipsub partial messages on the topic. Null when the tracer cannot see the partial messages subscription options' CODEC(ZSTD(1));

ALTER TABLE libp2p_rpc_meta_subscription ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS requests_partial Nullable(Bool)
        COMMENT 'Whether the peer asked to receive gossipsub partial messages on the topic. Null when the tracer cannot see the partial messages subscription options' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS supports_sending_partial Nullable(Bool)
        COMMENT 'Whether the peer can send gossipsub partial messages on the topic. Null when the tracer cannot see the partial messages subscription options' CODEC(ZSTD(1));
