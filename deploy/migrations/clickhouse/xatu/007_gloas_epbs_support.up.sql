-- EIP-7732 ePBS support: new tables + column additions
-- 10 new tables: 2 cannon, 4 sentry SSE, 4 libp2p gossip.
-- Plus column additions to existing tables.

---------------------------------------------------------------------
-- 1. CANNON: canonical_beacon_block_payload_attestation
--    Aggregated PTC attestations from block body (max 4/block)
---------------------------------------------------------------------
CREATE TABLE canonical_beacon_block_payload_attestation_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number of the block containing this payload attestation' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Root of the block containing this attestation' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'Block version (e.g. gloas)',
    `position` UInt32 COMMENT 'Position of the payload attestation in the block body (0-3)' CODEC(DoubleDelta, ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'The block root being attested to by the PTC' CODEC(ZSTD(1)),
    `payload_present` Bool COMMENT 'Whether the PTC attests payload was present' CODEC(ZSTD(1)),
    `blob_data_available` Bool COMMENT 'Whether the PTC attests blob data was available' CODEC(ZSTD(1)),
    `aggregation_bits` String COMMENT 'Bitvector of PTC members (512 bits) as hex' CODEC(ZSTD(1)),
    `attesting_validator_count` UInt32 COMMENT 'Number of PTC validators in this aggregation' CODEC(DoubleDelta, ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, block_root, position)
COMMENT 'Aggregated PTC payload attestations from canonical beacon blocks (max 4 per block).';

CREATE TABLE canonical_beacon_block_payload_attestation ON CLUSTER '{cluster}'
    AS canonical_beacon_block_payload_attestation_local
    ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_payload_attestation_local, rand());

---------------------------------------------------------------------
-- 2. CANNON: canonical_beacon_block_execution_payload_bid
--    Winning builder bid from each block (1 per block)
---------------------------------------------------------------------
CREATE TABLE canonical_beacon_block_execution_payload_bid_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number of the block containing this bid' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Root of the block containing this bid' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'Block version (e.g. gloas)',
    `builder_index` UInt64 COMMENT 'Index of the builder in the builder registry' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash committed to in the bid' CODEC(ZSTD(1)),
    `parent_block_hash` FixedString(66) COMMENT 'Parent execution block hash' CODEC(ZSTD(1)),
    `parent_block_root` FixedString(66) COMMENT 'Parent beacon block root' CODEC(ZSTD(1)),
    `value` UInt64 COMMENT 'Bid value in Gwei' CODEC(ZSTD(1)),
    `execution_payment` UInt64 COMMENT 'Execution payment in Gwei' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'Fee recipient address' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'Gas limit for the execution payload' CODEC(DoubleDelta, ZSTD(1)),
    `prev_randao` FixedString(66) COMMENT 'Previous RANDAO value' CODEC(ZSTD(1)),
    `blob_kzg_commitment_count` UInt32 COMMENT 'Number of blob KZG commitments' CODEC(DoubleDelta, ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, block_root)
COMMENT 'Winning execution payload bid from canonical beacon blocks (1 per block).';

CREATE TABLE canonical_beacon_block_execution_payload_bid ON CLUSTER '{cluster}'
    AS canonical_beacon_block_execution_payload_bid_local
    ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_execution_payload_bid_local, rand());

---------------------------------------------------------------------
-- 3. SENTRY SSE: beacon_api_eth_v1_events_execution_payload
--    Payload envelope arrival from beacon API SSE
---------------------------------------------------------------------
CREATE TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number from the event payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Beacon block root the envelope references' CODEC(ZSTD(1)),
    `builder_index` UInt64 COMMENT 'Index of the builder that produced the payload' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash' CODEC(ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'Execution state root' CODEC(ZSTD(1)),
    `slot_number` Nullable(UInt64) COMMENT 'EIP-7843 SLOTNUM: the execution payload slot_number field (typically equals the beacon slot)' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, block_root)
COMMENT 'Execution payload envelope arrivals from beacon API SSE (execution_payload event, fires on import into fork-choice).';

CREATE TABLE beacon_api_eth_v1_events_execution_payload ON CLUSTER '{cluster}'
    AS beacon_api_eth_v1_events_execution_payload_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_api_eth_v1_events_execution_payload_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, block_root));

---------------------------------------------------------------------
-- 3b. SENTRY SSE: beacon_api_eth_v1_events_payload_attestation
--     Individual PTC attestation from beacon API SSE (~512/slot)
---------------------------------------------------------------------
CREATE TABLE beacon_api_eth_v1_events_payload_attestation_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number from the event payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'Index of the PTC validator' CODEC(ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'Block root being attested to' CODEC(ZSTD(1)),
    `payload_present` Bool COMMENT 'Whether the validator attests payload was present' CODEC(ZSTD(1)),
    `blob_data_available` Bool COMMENT 'Whether the validator attests blob data was available' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, beacon_block_root, validator_index)
COMMENT 'Individual PTC payload attestation messages from beacon API SSE (payload_attestation_message event, ~512 per slot).';

CREATE TABLE beacon_api_eth_v1_events_payload_attestation ON CLUSTER '{cluster}'
    AS beacon_api_eth_v1_events_payload_attestation_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_api_eth_v1_events_payload_attestation_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, beacon_block_root, validator_index));

---------------------------------------------------------------------
-- 3c. SENTRY SSE: beacon_api_eth_v1_events_execution_payload_bid
--     Builder bid from beacon API SSE
---------------------------------------------------------------------
CREATE TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number from the event payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `builder_index` UInt64 COMMENT 'Index of the builder' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash committed to in the bid' CODEC(ZSTD(1)),
    `parent_block_hash` FixedString(66) COMMENT 'Parent execution block hash' CODEC(ZSTD(1)),
    `parent_block_root` FixedString(66) COMMENT 'Parent beacon block root' CODEC(ZSTD(1)),
    `value` UInt64 COMMENT 'Bid value in Gwei' CODEC(ZSTD(1)),
    `execution_payment` UInt64 COMMENT 'Execution payment in Gwei' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'Fee recipient address' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'Gas limit' CODEC(DoubleDelta, ZSTD(1)),
    `blob_kzg_commitment_count` UInt32 COMMENT 'Number of blob KZG commitments' CODEC(DoubleDelta, ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, block_hash)
COMMENT 'Builder bids from beacon API SSE (execution_payload_bid event).';

CREATE TABLE beacon_api_eth_v1_events_execution_payload_bid ON CLUSTER '{cluster}'
    AS beacon_api_eth_v1_events_execution_payload_bid_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_api_eth_v1_events_execution_payload_bid_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, block_hash));

---------------------------------------------------------------------
-- 3d. SENTRY SSE: beacon_api_eth_v1_events_proposer_preferences
--     Proposer preferences from beacon API SSE
---------------------------------------------------------------------
CREATE TABLE beacon_api_eth_v1_events_proposer_preferences_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Proposal slot' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `validator_index` UInt32 COMMENT 'Index of the proposing validator' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'Preferred fee recipient address' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'Preferred gas limit' CODEC(DoubleDelta, ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, validator_index)
COMMENT 'Proposer preferences from beacon API SSE (proposer_preferences event).';

CREATE TABLE beacon_api_eth_v1_events_proposer_preferences ON CLUSTER '{cluster}'
    AS beacon_api_eth_v1_events_proposer_preferences_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_api_eth_v1_events_proposer_preferences_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, validator_index));

---------------------------------------------------------------------
-- 3e. SENTRY SSE: beacon_api_eth_v1_events_execution_payload_gossip
--     Full envelope on first gossip (analog of block_gossip).
--     Fires earlier than execution_payload (which fires on import).
---------------------------------------------------------------------
CREATE TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number from the event payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Beacon block root the envelope references' CODEC(ZSTD(1)),
    `builder_index` UInt64 COMMENT 'Index of the builder that produced the payload' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash' CODEC(ZSTD(1)),
    `state_root` FixedString(66) COMMENT 'Execution state root' CODEC(ZSTD(1)),
    `slot_number` Nullable(UInt64) COMMENT 'EIP-7843 SLOTNUM: the execution payload slot_number field (typically equals the beacon slot)' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, block_root)
COMMENT 'Execution payload envelope first-seen-on-gossip arrivals from beacon API SSE (execution_payload_gossip event, fires before import).';

CREATE TABLE beacon_api_eth_v1_events_execution_payload_gossip ON CLUSTER '{cluster}'
    AS beacon_api_eth_v1_events_execution_payload_gossip_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_api_eth_v1_events_execution_payload_gossip_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, block_root));

---------------------------------------------------------------------
-- 3f. SENTRY SSE: beacon_api_eth_v1_events_execution_payload_available
--     Lightweight signal that payload+blobs are locally available
--     for PTC vote (block_root + slot only).
---------------------------------------------------------------------
CREATE TABLE beacon_api_eth_v1_events_execution_payload_available_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry received the event from a beacon node' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot of the block whose payload is now available' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Beacon block root whose payload+blobs are locally available' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, block_root)
COMMENT 'Execution payload availability signals from beacon API SSE (execution_payload_available event, fires when payload+blobs are locally verified for PTC vote).';

CREATE TABLE beacon_api_eth_v1_events_execution_payload_available ON CLUSTER '{cluster}'
    AS beacon_api_eth_v1_events_execution_payload_available_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_api_eth_v1_events_execution_payload_available_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, block_root));

---------------------------------------------------------------------
-- 4. LIBP2P: libp2p_gossipsub_execution_payload_envelope
---------------------------------------------------------------------
CREATE TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - propagation_slot_start_diff COMMENT 'Version for dedup: prefer lowest propagation time' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start time of the wall clock slot' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start time of the wall clock epoch' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Propagation delay from slot start in ms' CODEC(ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Beacon block root the envelope references' CODEC(ZSTD(1)),
    `builder_index` UInt64 COMMENT 'Index of the builder that produced the payload' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id)
COMMENT 'Execution payload envelope gossip propagation from libp2p.';

CREATE TABLE libp2p_gossipsub_execution_payload_envelope ON CLUSTER '{cluster}'
    AS libp2p_gossipsub_execution_payload_envelope_local
    ENGINE = Distributed('{cluster}', currentDatabase(), libp2p_gossipsub_execution_payload_envelope_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id));

---------------------------------------------------------------------
-- 5. LIBP2P: libp2p_gossipsub_execution_payload_bid
---------------------------------------------------------------------
CREATE TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - propagation_slot_start_diff COMMENT 'Version for dedup: prefer lowest propagation time' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start time of the wall clock slot' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start time of the wall clock epoch' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Propagation delay from slot start in ms' CODEC(ZSTD(1)),
    `builder_index` UInt64 COMMENT 'Index of the builder' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash committed to in the bid' CODEC(ZSTD(1)),
    `parent_block_hash` FixedString(66) COMMENT 'Parent execution block hash' CODEC(ZSTD(1)),
    `value` UInt64 COMMENT 'Bid value in Gwei' CODEC(ZSTD(1)),
    `execution_payment` UInt64 COMMENT 'Execution payment in Gwei' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'Fee recipient address' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'Gas limit' CODEC(DoubleDelta, ZSTD(1)),
    `blob_kzg_commitment_count` UInt32 COMMENT 'Number of blob KZG commitments' CODEC(DoubleDelta, ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id)
COMMENT 'Builder bid gossip propagation from libp2p.';

CREATE TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    AS libp2p_gossipsub_execution_payload_bid_local
    ENGINE = Distributed('{cluster}', currentDatabase(), libp2p_gossipsub_execution_payload_bid_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id));

---------------------------------------------------------------------
-- 6. LIBP2P: libp2p_gossipsub_payload_attestation_message
--    ~512 per slot, high volume
---------------------------------------------------------------------
CREATE TABLE libp2p_gossipsub_payload_attestation_message_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - propagation_slot_start_diff COMMENT 'Version for dedup: prefer lowest propagation time' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start time of the wall clock slot' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start time of the wall clock epoch' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Propagation delay from slot start in ms' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'Index of the PTC validator' CODEC(ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'Block root being attested to' CODEC(ZSTD(1)),
    `payload_present` Bool COMMENT 'Whether the validator attests payload was present' CODEC(ZSTD(1)),
    `blob_data_available` Bool COMMENT 'Whether the validator attests blob data was available' CODEC(ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id)
COMMENT 'Individual PTC payload attestation messages from libp2p gossip (~512 per slot).';

CREATE TABLE libp2p_gossipsub_payload_attestation_message ON CLUSTER '{cluster}'
    AS libp2p_gossipsub_payload_attestation_message_local
    ENGINE = Distributed('{cluster}', currentDatabase(), libp2p_gossipsub_payload_attestation_message_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id));

---------------------------------------------------------------------
-- 7. LIBP2P: libp2p_gossipsub_proposer_preferences
---------------------------------------------------------------------
CREATE TABLE libp2p_gossipsub_proposer_preferences_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - propagation_slot_start_diff COMMENT 'Version for dedup: prefer lowest propagation time' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'Timestamp of the event with millisecond precision' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Proposal slot' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'Start date and time of the slot' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'Start date and time of the epoch' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot` UInt32 COMMENT 'Wall clock slot when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_slot_start_date_time` DateTime COMMENT 'Start time of the wall clock slot' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch` UInt32 COMMENT 'Wall clock epoch when the event was received' CODEC(DoubleDelta, ZSTD(1)),
    `wallclock_epoch_start_date_time` DateTime COMMENT 'Start time of the wall clock epoch' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Propagation delay from slot start in ms' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'Index of the proposing validator' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'Preferred fee recipient address' CODEC(ZSTD(1)),
    `gas_limit` UInt64 COMMENT 'Preferred gas limit' CODEC(DoubleDelta, ZSTD(1)),
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
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toYYYYMM(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id)
COMMENT 'Proposer preferences gossip propagation from libp2p.';

CREATE TABLE libp2p_gossipsub_proposer_preferences ON CLUSTER '{cluster}'
    AS libp2p_gossipsub_proposer_preferences_local
    ENGINE = Distributed('{cluster}', currentDatabase(), libp2p_gossipsub_proposer_preferences_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, peer_id_unique_key, message_id));

---------------------------------------------------------------------
-- 8. ALTER: Add ePBS columns to beacon block tables
---------------------------------------------------------------------
ALTER TABLE canonical_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `builder_index` Nullable(UInt64) COMMENT 'Builder index from the bid (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payload_block_access_list_root,
    ADD COLUMN IF NOT EXISTS `bid_value` Nullable(UInt64) COMMENT 'Bid value in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER builder_index,
    ADD COLUMN IF NOT EXISTS `execution_payment` Nullable(UInt64) COMMENT 'Execution payment in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER bid_value,
    ADD COLUMN IF NOT EXISTS `payload_present` Nullable(Bool) COMMENT 'Whether execution payload was delivered (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payment;

ALTER TABLE canonical_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `builder_index` Nullable(UInt64) COMMENT 'Builder index from the bid (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payload_block_access_list_root,
    ADD COLUMN IF NOT EXISTS `bid_value` Nullable(UInt64) COMMENT 'Bid value in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER builder_index,
    ADD COLUMN IF NOT EXISTS `execution_payment` Nullable(UInt64) COMMENT 'Execution payment in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER bid_value,
    ADD COLUMN IF NOT EXISTS `payload_present` Nullable(Bool) COMMENT 'Whether execution payload was delivered (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payment;

ALTER TABLE beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `builder_index` Nullable(UInt64) COMMENT 'Builder index from the bid (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payload_slot_number,
    ADD COLUMN IF NOT EXISTS `bid_value` Nullable(UInt64) COMMENT 'Bid value in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER builder_index,
    ADD COLUMN IF NOT EXISTS `execution_payment` Nullable(UInt64) COMMENT 'Execution payment in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER bid_value,
    ADD COLUMN IF NOT EXISTS `payload_present` Nullable(Bool) COMMENT 'Whether execution payload was delivered (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payment;

ALTER TABLE beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `builder_index` Nullable(UInt64) COMMENT 'Builder index from the bid (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payload_slot_number,
    ADD COLUMN IF NOT EXISTS `bid_value` Nullable(UInt64) COMMENT 'Bid value in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER builder_index,
    ADD COLUMN IF NOT EXISTS `execution_payment` Nullable(UInt64) COMMENT 'Execution payment in Gwei (Gloas+)' CODEC(ZSTD(1)) AFTER bid_value,
    ADD COLUMN IF NOT EXISTS `payload_present` Nullable(Bool) COMMENT 'Whether execution payload was delivered (Gloas+)' CODEC(ZSTD(1)) AFTER execution_payment;

---------------------------------------------------------------------
-- 8b. ALTER: Add withdrawal_type column to canonical withdrawal tables
--    EIP-7732 introduces "builder" withdrawals (validator_index >= 2^40).
--    Pre-Gloas rows: empty string. Gloas+: "validator" or "builder".
---------------------------------------------------------------------
ALTER TABLE canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `withdrawal_type` LowCardinality(String) DEFAULT '' COMMENT 'Classification of the withdrawal recipient (Gloas+: validator|builder, pre-Gloas: empty)' CODEC(ZSTD(1));

ALTER TABLE canonical_beacon_block_withdrawal ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `withdrawal_type` LowCardinality(String) DEFAULT '' COMMENT 'Classification of the withdrawal recipient (Gloas+: validator|builder, pre-Gloas: empty)' CODEC(ZSTD(1));

---------------------------------------------------------------------
-- 004_gloas_synthetic_events
--
-- EIP-7732 ePBS: synthesized observability events from beacon-node internals
-- (TYSM-instrumented). No beacon API equivalent today.
--
-- These events fire on every TYSM-patched beacon node (multi-witness) — the
-- ORDER BY / Distributed sharding keep meta_client_name in the composite so
-- per-node observations aren't collapsed.
---------------------------------------------------------------------

---------------------------------------------------------------------
-- 1. beacon_synthetic_payload_status_resolved
--    Fork-choice slot payload status transitions (PENDING/FULL/EMPTY/INVALID).
---------------------------------------------------------------------
CREATE TABLE beacon_synthetic_payload_status_resolved_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the beacon node resolved the status (TYSM ResolvedAt)' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot number whose payload status was resolved' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between event_date_time and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'Beacon block root for this slot' CODEC(ZSTD(1)),
    `block_hash` FixedString(66) COMMENT 'Execution block hash (when known)' CODEC(ZSTD(1)),
    `status` LowCardinality(String) COMMENT 'New status: PENDING / FULL / EMPTY / INVALID',
    `previous_status` LowCardinality(String) COMMENT 'Previous status before this transition',
    `payload_timeliness_votes_positive` UInt64 COMMENT 'Count of PTC votes with payload_present=true' CODEC(ZSTD(1)),
    `payload_timeliness_votes_negative` Nullable(UInt64) COMMENT 'Count of PTC votes with payload_present=false (explicit negative). NULL when CL does not surface three-state PR #5180 breakdown' CODEC(ZSTD(1)),
    `payload_timeliness_votes_absent` Nullable(UInt64) COMMENT 'Count of PTC seats with no vote (Optional[bool]==None per PR #5180). NULL when CL does not surface the breakdown' CODEC(ZSTD(1)),
    `data_available_votes_positive` UInt64 COMMENT 'Count of PTC votes with blob_data_available=true' CODEC(ZSTD(1)),
    `data_available_votes_negative` Nullable(UInt64) COMMENT 'Count of PTC votes with blob_data_available=false (explicit negative). NULL when CL does not surface three-state PR #5180 breakdown' CODEC(ZSTD(1)),
    `data_available_votes_absent` Nullable(UInt64) COMMENT 'Count of PTC seats with no data-availability vote (Optional[bool]==None per PR #5180). NULL when CL does not surface the breakdown' CODEC(ZSTD(1)),
    `ptc_size` UInt64 COMMENT 'Total PTC committee size (typically 512)' CODEC(ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, block_root, status)
COMMENT 'Fork-choice payload status transitions (EIP-7732 ePBS) synthesized from TYSM-instrumented beacon node internals. Multi-witness (per-node).';

CREATE TABLE beacon_synthetic_payload_status_resolved ON CLUSTER '{cluster}'
    AS beacon_synthetic_payload_status_resolved_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_synthetic_payload_status_resolved_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, block_root, status));

---------------------------------------------------------------------
-- 2. beacon_synthetic_builder_pending_payment_settlement
--    Epoch-boundary builder pending payment settle/drop decisions.
---------------------------------------------------------------------
CREATE TABLE beacon_synthetic_builder_pending_payment_settlement_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the beacon node processed this settlement (TYSM ResolvedAt)' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch boundary at which this settlement was processed' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `builder_index` UInt64 COMMENT 'Index of the builder in the builder registry' CODEC(ZSTD(1)),
    `fee_recipient` FixedString(42) COMMENT 'Builder fee recipient address' CODEC(ZSTD(1)),
    `amount` UInt64 COMMENT 'Payment amount in Gwei' CODEC(ZSTD(1)),
    `weight` UInt64 COMMENT 'Quorum weight achieved in Gwei' CODEC(ZSTD(1)),
    `quorum` UInt64 COMMENT 'Quorum threshold needed in Gwei' CODEC(ZSTD(1)),
    `outcome` LowCardinality(String) COMMENT 'Settlement outcome: SETTLED / DROPPED',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY (epoch_start_date_time, meta_network_name, meta_client_name, builder_index, outcome)
COMMENT 'Builder pending payment settle/drop decisions at epoch boundary (EIP-7732 ePBS) synthesized from TYSM-instrumented beacon node internals. Multi-witness (per-node).';

CREATE TABLE beacon_synthetic_builder_pending_payment_settlement ON CLUSTER '{cluster}'
    AS beacon_synthetic_builder_pending_payment_settlement_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_synthetic_builder_pending_payment_settlement_local,
        cityHash64(epoch_start_date_time, meta_network_name, meta_client_name, builder_index, outcome));

---------------------------------------------------------------------
-- 3. beacon_synthetic_payload_attestation_processed
--    Per-PTC-vote enrichment: fires after a payload_attestation_message has
--    passed full gossip validation (signature, validator-in-PTC, block-root
--    seen + valid, slot-current, first-from-this-validator dedup) and been
--    committed for downstream pipeline use. Counterpart to the gossip-receipt
--    `beacon_api_eth_v1_events_payload_attestation`: observing both gives a
--    per-validator picture of validation latency and whether the vote was
--    admitted vs dropped at some validation stage.
---------------------------------------------------------------------
CREATE TABLE beacon_synthetic_payload_attestation_processed_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the beacon node processed this PTC vote (TYSM ProcessedAt)' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'Slot the PTC vote applies to' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `propagation_slot_start_diff` UInt32 COMMENT 'Difference between processed_at and slot_start_date_time in ms' CODEC(ZSTD(1)),
    `epoch` UInt32 COMMENT 'Epoch number' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `beacon_block_root` FixedString(66) COMMENT 'Beacon block root the PTC validator attested to' CODEC(ZSTD(1)),
    `validator_index` UInt32 COMMENT 'Index of the PTC validator' CODEC(ZSTD(1)),
    `payload_present` Bool COMMENT 'Whether the validator attests payload was present' CODEC(ZSTD(1)),
    `blob_data_available` Bool COMMENT 'Whether the validator attests blob data was available' CODEC(ZSTD(1)),
    `peer_id` String COMMENT 'Peer ID we received this PTC vote from on the gossip wire' CODEC(ZSTD(1)),
    `processing_duration_ms` UInt64 COMMENT 'Time from gossip receipt to processing completion in milliseconds' CODEC(ZSTD(1)),
    `received_at` DateTime64(3) COMMENT 'Wall-clock time the PTC vote was first received from gossip' CODEC(DoubleDelta, ZSTD(1)),
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_consensus_version` LowCardinality(String) COMMENT 'Consensus client version',
    `meta_consensus_version_major` LowCardinality(String) COMMENT 'Consensus client major version',
    `meta_consensus_version_minor` LowCardinality(String) COMMENT 'Consensus client minor version',
    `meta_consensus_version_patch` LowCardinality(String) COMMENT 'Consensus client patch version',
    `meta_consensus_implementation` LowCardinality(String) COMMENT 'Consensus client implementation',
    `meta_labels` Map(String, String) COMMENT 'Labels associated with the event' CODEC(ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
) PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name, beacon_block_root, validator_index)
COMMENT 'PTC votes after full gossip validation completed (EIP-7732 ePBS) synthesized from TYSM-instrumented beacon node internals. Enrichment counterpart to beacon_api_eth_v1_events_payload_attestation. Multi-witness (per-node).';

CREATE TABLE beacon_synthetic_payload_attestation_processed ON CLUSTER '{cluster}'
    AS beacon_synthetic_payload_attestation_processed_local
    ENGINE = Distributed('{cluster}', currentDatabase(), beacon_synthetic_payload_attestation_processed_local,
        cityHash64(slot_start_date_time, meta_network_name, meta_client_name, beacon_block_root, validator_index));
