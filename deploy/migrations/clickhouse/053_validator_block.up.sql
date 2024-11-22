-- Step 1: Create the new table.
CREATE TABLE default.beacon_api_eth_v3_validator_block_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number within the payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the reorg slot started',
    `epoch` UInt32 COMMENT 'The epoch number in the beacon API event stream payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `block_total_bytes` Nullable(UInt32) COMMENT 'The total bytes of the beacon block payload' CODEC(ZSTD(1)),
    `block_total_bytes_compressed` Nullable(UInt32) COMMENT 'The total bytes of the beacon block payload when compressed using snappy' CODEC(ZSTD(1)),
    `consensus_payload_value` Nullable(UInt64) COMMENT 'Consensus rewards paid to the proposer for this block, in Wei. Use to determine relative value of consensus blocks.' CODEC(ZSTD(1)),
    `execution_payload_value` Nullable(UInt64) COMMENT 'Execution payload value in Wei. Use to determine relative value of execution payload.' CODEC(ZSTD(1)),
    `execution_payload_block_number` UInt32 COMMENT 'The block number of the execution payload',
    `execution_payload_base_fee_per_gas` Nullable(UInt128) COMMENT 'Base fee per gas for execution payload' CODEC(ZSTD(1)),
    `execution_payload_blob_gas_used` Nullable(UInt64) COMMENT 'Gas used for blobs in execution payload' CODEC(ZSTD(1)),
    `execution_payload_excess_blob_gas` Nullable(UInt64) COMMENT 'Excess gas used for blobs in execution payload' CODEC(ZSTD(1)),
    `execution_payload_gas_limit` Nullable(UInt64) COMMENT 'Gas limit for execution payload' CODEC(DoubleDelta, ZSTD(1)),
    `execution_payload_gas_used` Nullable(UInt64) COMMENT 'Gas used for execution payload' CODEC(ZSTD(1)),
    `execution_payload_transactions_count` Nullable(UInt32) COMMENT 'The transaction count of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_transactions_total_bytes` Nullable(UInt32) COMMENT 'The transaction total bytes of the execution payload' CODEC(ZSTD(1)),
    `execution_payload_transactions_total_bytes_compressed` Nullable(UInt32) COMMENT 'The transaction total bytes of the execution payload when compressed using snappy' CODEC(ZSTD(1)),
    `meta_client_name` LowCardinality(String) COMMENT 'Name of the client that generated the event',
    `meta_client_id` String COMMENT 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.' CODEC(ZSTD(1)),
    `meta_client_version` LowCardinality(String) COMMENT 'Version of the client that generated the event',
    `meta_client_implementation` LowCardinality(String) COMMENT 'Implementation of the client that generated the event',
    `meta_client_os` LowCardinality(String) COMMENT 'Operating system of the client that generated the event',
    `meta_client_ip` Nullable(IPv6) COMMENT 'IP address of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_city` LowCardinality(String) COMMENT 'City of the client that generated the event',
    `meta_client_geo_country` LowCardinality(String) COMMENT 'Country of the client that generated the event',
    `meta_client_geo_country_code` LowCardinality(String) COMMENT 'Country code of the client that generated the event',
    `meta_client_geo_continent_code` LowCardinality(String) COMMENT 'Continent code of the client that generated the event',
    `meta_client_geo_longitude` Nullable(Float64) COMMENT 'Longitude of the client that generated the event',
    `meta_client_geo_latitude` Nullable(Float64) COMMENT 'Latitude of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_number` Nullable(UInt32) COMMENT 'Autonomous system number of the client that generated the event' CODEC(ZSTD(1)),
    `meta_client_geo_autonomous_system_organization` Nullable(String) COMMENT 'Autonomous system organization of the client that generated the event' CODEC(ZSTD(1)),
    `meta_network_id` Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Ethereum consensus client version that generated the event',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Ethereum consensus client major version that generated the event',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Ethereum consensus client minor version that generated the event',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Ethereum consensus client patch version that generated the event',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Ethereum consensus client implementation that generated the event',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY
(
    event_date_time,
    slot_start_date_time,
    meta_network_name,
    meta_client_name
)
COMMENT 'Contains beacon API /eth/v3/validator/blocks/{slot} data from each sentry client attached to a beacon node.';

-- Step 2: Create the distributed table.
CREATE TABLE default.beacon_api_eth_v3_validator_block ON CLUSTER '{cluster}' AS default.beacon_api_eth_v3_validator_block_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v3_validator_block_local,
    cityHash64(
        event_date_time,
        slot_start_date_time,
        meta_network_name,
        meta_client_name
    )
);
