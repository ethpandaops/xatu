CREATE TABLE default.mev_relay_proposer_payload_delivered_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the payload was delivered' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number within the payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The start time for the slot that the bid is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number derived from the slot that the bid is for' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The start time for the epoch that the bid is for' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'The wallclock slot when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'The start time for the slot when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'The wallclock epoch when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'The start time for the wallclock epoch when the request was sent' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number of the payload' CODEC(DoubleDelta, ZSTD(1)),
    `relay_name` String COMMENT 'The relay that delivered the payload' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'The block hash associated with the payload' CODEC(ZSTD(1)),
    `proposer_pubkey` String COMMENT 'The proposer pubkey that received the payload' CODEC(ZSTD(1)),
    `builder_pubkey` String COMMENT 'The builder pubkey that sent the payload' CODEC(ZSTD(1)),
    `proposer_fee_recipient` FixedString(42) COMMENT 'The proposer fee recipient of the payload' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'The gas limit of the payload' CODEC(DoubleDelta, ZSTD(1)),
    `gas_used` UInt64 COMMENT 'The gas used by the payload' CODEC(DoubleDelta, ZSTD(1)),
    `num_tx` UInt32 COMMENT 'The number of transactions in the payload' CODEC(DoubleDelta, ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
    (
        slot_start_date_time,
        meta_network_name,
        slot,
        block_hash,
        meta_client_name,
        builder_pubkey,
        proposer_pubkey
    ) COMMENT 'Contains MEV relay proposer payload delivered data.';

CREATE TABLE default.mev_relay_proposer_payload_delivered ON CLUSTER '{cluster}' AS default.mev_relay_proposer_payload_delivered_local ENGINE = Distributed(
    '{cluster}',
    default,
    mev_relay_proposer_payload_delivered_local,
    cityHash64(
        slot,
        meta_network_name
    )
);