-- Add execution_payload_slot_number to beacon_api_eth_v2_beacon_block
ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS execution_payload_slot_number Nullable(UInt64)
        CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas;

ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS execution_payload_slot_number Nullable(UInt64)
        CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas;

-- Add execution_payload_slot_number and execution_payload_block_access_list_root to canonical_beacon_block
ALTER TABLE default.canonical_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS execution_payload_slot_number Nullable(UInt64)
        CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas,
    ADD COLUMN IF NOT EXISTS execution_payload_block_access_list_root Nullable(FixedString(66))
        CODEC(ZSTD(1)) AFTER execution_payload_slot_number;

ALTER TABLE default.canonical_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS execution_payload_slot_number Nullable(UInt64)
        CODEC(DoubleDelta, ZSTD(1)) AFTER execution_payload_excess_blob_gas,
    ADD COLUMN IF NOT EXISTS execution_payload_block_access_list_root Nullable(FixedString(66))
        CODEC(ZSTD(1)) AFTER execution_payload_slot_number;

-- Create canonical_beacon_block_access_list table
CREATE TABLE IF NOT EXISTS default.canonical_beacon_block_access_list_local ON CLUSTER '{cluster}' (
    updated_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    slot UInt32 CODEC(DoubleDelta, ZSTD(1)),
    slot_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    epoch UInt32 CODEC(DoubleDelta, ZSTD(1)),
    epoch_start_date_time DateTime CODEC(DoubleDelta, ZSTD(1)),
    block_root FixedString(66) CODEC(ZSTD(1)),
    block_number UInt64 CODEC(DoubleDelta, ZSTD(1)),
    block_hash FixedString(66) CODEC(ZSTD(1)),
    address FixedString(42) CODEC(ZSTD(1)),
    change_type LowCardinality(String) CODEC(ZSTD(1)),
    block_access_index UInt32 CODEC(DoubleDelta, ZSTD(1)),
    storage_key FixedString(66) CODEC(ZSTD(1)),
    new_value Nullable(String) CODEC(ZSTD(1)),
    meta_client_name LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_id String CODEC(ZSTD(1)),
    meta_client_version LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_implementation LowCardinality(String) CODEC(ZSTD(1)),
    meta_client_os LowCardinality(String) CODEC(ZSTD(1)),
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
    meta_network_name LowCardinality(String) CODEC(ZSTD(1)),
    meta_consensus_version LowCardinality(String) CODEC(ZSTD(1)),
    meta_consensus_implementation LowCardinality(String) CODEC(ZSTD(1)),
    meta_labels Map(String, String) CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(updated_date_time)
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, block_hash, address, change_type, storage_key, block_access_index);

CREATE TABLE IF NOT EXISTS default.canonical_beacon_block_access_list ON CLUSTER '{cluster}'
    AS default.canonical_beacon_block_access_list_local
    ENGINE = Distributed('{cluster}', default, canonical_beacon_block_access_list_local, rand());
