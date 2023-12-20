CREATE TABLE beacon_api_eth_v1_events_head_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
  block FixedString(66) Codec(ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  epoch_transition Bool,
  execution_optimistic Bool,
  previous_duty_dependent_root FixedString(66) Codec(ZSTD(1)),
  current_duty_dependent_root FixedString(66) Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_slot (SELECT * ORDER BY slot),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_block (SELECT * ORDER BY block)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_head on cluster '{cluster}' AS beacon_api_eth_v1_events_head_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_head_local, rand());

CREATE TABLE beacon_api_eth_v1_events_block_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
  block FixedString(66) Codec(ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  execution_optimistic Bool,
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_slot (SELECT * ORDER BY slot),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_block (SELECT * ORDER BY block)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_block on cluster '{cluster}' AS beacon_api_eth_v1_events_block_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_block_local, rand());

CREATE TABLE beacon_api_eth_v1_events_attestation_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
  committee_index LowCardinality(String),
  signature String Codec(ZSTD(1)),
  aggregation_bits String Codec(ZSTD(1)),
  beacon_block_root FixedString(66) Codec(ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  source_epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  source_epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  source_root FixedString(66) Codec(ZSTD(1)),
  target_epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  target_epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  target_root FixedString(66) Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_slot (SELECT * ORDER BY slot),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_beacon_block_root (SELECT * ORDER BY beacon_block_root)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_attestation on cluster '{cluster}' AS beacon_api_eth_v1_events_attestation_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_attestation_local, rand());

CREATE TABLE beacon_api_eth_v1_events_voluntary_exit_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  validator_index UInt32 Codec(ZSTD(1)),
  signature String Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_validator_index (SELECT * ORDER BY validator_index)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY (epoch_start_date_time, meta_network_name, meta_client_name)
TTL epoch_start_date_time TO VOLUME 'default',
    epoch_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    epoch_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    epoch_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    epoch_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_voluntary_exit on cluster '{cluster}' AS beacon_api_eth_v1_events_voluntary_exit_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_voluntary_exit_local, rand());

CREATE TABLE beacon_api_eth_v1_events_finalized_checkpoint_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  block FixedString(66) Codec(ZSTD(1)),
  state FixedString(66) Codec(ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  execution_optimistic Bool,
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_block (SELECT * ORDER BY block),
  PROJECTION projection_state (SELECT * ORDER BY state)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(epoch_start_date_time)
ORDER BY (epoch_start_date_time, meta_network_name, meta_client_name)
TTL epoch_start_date_time TO VOLUME 'default',
    epoch_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    epoch_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    epoch_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    epoch_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_finalized_checkpoint on cluster '{cluster}' AS beacon_api_eth_v1_events_finalized_checkpoint_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_finalized_checkpoint_local, rand());

CREATE TABLE beacon_api_eth_v1_events_chain_reorg_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
  depth UInt16 Codec(DoubleDelta, ZSTD(1)),
  old_head_block FixedString(66) Codec(ZSTD(1)),
  new_head_block FixedString(66) Codec(ZSTD(1)),
  old_head_state FixedString(66) Codec(ZSTD(1)),
  new_head_state FixedString(66) Codec(ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  execution_optimistic Bool,
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_slot (SELECT * ORDER BY slot),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_new_head_state (SELECT * ORDER BY new_head_state),
  PROJECTION projection_old_head_state (SELECT * ORDER BY old_head_state),
  PROJECTION projection_new_head_block (SELECT * ORDER BY new_head_block),
  PROJECTION projection_old_head_block (SELECT * ORDER BY old_head_block)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_chain_reorg on cluster '{cluster}' AS beacon_api_eth_v1_events_chain_reorg_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_chain_reorg_local, rand());

CREATE TABLE beacon_api_eth_v1_events_contribution_and_proof_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  aggregator_index UInt32 Codec(ZSTD(1)),
  contribution_slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  contribution_slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  contribution_propagation_slot_start_diff UInt32 Codec(ZSTD(1)),
  contribution_beacon_block_root FixedString(66) Codec(ZSTD(1)),
  contribution_subcommittee_index LowCardinality(String),
  contribution_aggregation_bits String Codec(ZSTD(1)),
  contribution_signature String Codec(ZSTD(1)),
  contribution_epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  contribution_epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  selection_proof String Codec(ZSTD(1)),
  signature String Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  meta_consensus_version_major LowCardinality(String),
  meta_consensus_version_minor LowCardinality(String),
  meta_consensus_version_patch LowCardinality(String),
  meta_consensus_implementation LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_contribution_slot (SELECT * ORDER BY contribution_slot),
  PROJECTION projection_contribution_epoch (SELECT * ORDER BY contribution_epoch),
  PROJECTION projection_contribution_beacon_block_root (SELECT * ORDER BY contribution_beacon_block_root)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(contribution_slot_start_date_time)
ORDER BY (contribution_slot_start_date_time, meta_network_name, meta_client_name)
TTL contribution_slot_start_date_time TO VOLUME 'default',
    contribution_slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    contribution_slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    contribution_slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    contribution_slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v1_events_contribution_and_proof on cluster '{cluster}' AS beacon_api_eth_v1_events_contribution_and_proof_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v1_events_contribution_and_proof_local, rand());

CREATE TABLE beacon_api_slot_local on cluster '{cluster}' (
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_network_name LowCardinality(String),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_consensus_implementation LowCardinality(String),
  meta_consensus_version LowCardinality(String),
  blocks AggregateFunction(sum, UInt16) Codec(ZSTD(1)),
  attestations AggregateFunction(sum, UInt32) Codec(ZSTD(1)),
  PROJECTION projection_slot (SELECT * ORDER BY slot),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, slot, meta_network_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE MATERIALIZED VIEW beacon_api_slot_block_mv_local on cluster '{cluster}'
TO beacon_api_slot_local
AS SELECT
  slot,
  slot_start_date_time,
  epoch,
  epoch_start_date_time,
  meta_client_name,
  meta_network_name,
  meta_client_geo_city,
  meta_client_geo_continent_code,
  meta_client_geo_longitude,
  meta_client_geo_latitude,
  meta_consensus_implementation,
  meta_consensus_version,
  sumState(toUInt16(1)) AS blocks
FROM beacon_api_eth_v1_events_block_local
GROUP BY slot, slot_start_date_time, epoch, epoch_start_date_time, meta_client_name, meta_network_name, meta_client_geo_city, meta_client_geo_continent_code, meta_client_geo_longitude, meta_client_geo_latitude, meta_consensus_implementation, meta_consensus_version;

CREATE MATERIALIZED VIEW beacon_api_slot_attestation_mv_local on cluster '{cluster}'
TO beacon_api_slot_local
AS SELECT
  slot,
  slot_start_date_time,
  epoch,
  epoch_start_date_time,
  meta_client_name,
  meta_network_name,
  meta_client_geo_city,
  meta_client_geo_continent_code,
  meta_client_geo_longitude,
  meta_client_geo_latitude,
  meta_consensus_implementation,
  meta_consensus_version,
  sumState(toUInt32(1)) AS attestations
FROM beacon_api_eth_v1_events_attestation_local
GROUP BY slot, slot_start_date_time, epoch, epoch_start_date_time, meta_client_name, meta_network_name, meta_client_geo_city, meta_client_geo_continent_code, meta_client_geo_longitude, meta_client_geo_latitude, meta_consensus_implementation, meta_consensus_version;

CREATE TABLE beacon_api_slot on cluster '{cluster}' AS beacon_api_slot_local
ENGINE = Distributed('{cluster}', default, beacon_api_slot_local, rand());

CREATE TABLE mempool_transaction_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  hash FixedString(66) Codec(ZSTD(1)),
  from FixedString(42) Codec(ZSTD(1)),
  to FixedString(42) Codec(ZSTD(1)),
  nonce UInt64 Codec(ZSTD(1)),
  gas_price UInt128 Codec(ZSTD(1)),
  gas UInt64 Codec(ZSTD(1)),
  value UInt128 Codec(ZSTD(1)),
  size UInt32 Codec(ZSTD(1)),
  call_data_size UInt32 Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_execution_fork_id_hash LowCardinality(String),
  meta_execution_fork_id_next LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_hash (SELECT * ORDER BY hash),
  PROJECTION projection_from (SELECT * ORDER BY from),
  PROJECTION projection_to (SELECT * ORDER BY to)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(event_date_time)
ORDER BY (event_date_time, meta_network_name, meta_client_name)
TTL toDateTime(event_date_time) TO VOLUME 'default',
    toDateTime(event_date_time) + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    toDateTime(event_date_time) + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    toDateTime(event_date_time) + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    toDateTime(event_date_time) + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE mempool_transaction on cluster '{cluster}' AS mempool_transaction_local
ENGINE = Distributed('{cluster}', default, mempool_transaction_local, rand());

CREATE TABLE beacon_api_eth_v2_beacon_block_local on cluster '{cluster}' (
  event_date_time DateTime64(3) Codec(DoubleDelta, ZSTD(1)),
  slot UInt32 Codec(DoubleDelta, ZSTD(1)),
  slot_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  epoch UInt32 Codec(DoubleDelta, ZSTD(1)),
  epoch_start_date_time DateTime Codec(DoubleDelta, ZSTD(1)),
  block_root FixedString(66) Codec(ZSTD(1)),
  parent_root FixedString(66) Codec(ZSTD(1)),
  state_root FixedString(66) Codec(ZSTD(1)),
  proposer_index UInt32 Codec(ZSTD(1)),
  eth1_data_block_hash FixedString(66) Codec(ZSTD(1)),
  eth1_data_deposit_root FixedString(66) Codec(ZSTD(1)),
  execution_payload_block_hash FixedString(66) Codec(ZSTD(1)),
  execution_payload_block_number UInt32 Codec(DoubleDelta, ZSTD(1)),
  execution_payload_fee_recipient String Codec(ZSTD(1)),
  execution_payload_state_root FixedString(66) Codec(ZSTD(1)),
  execution_payload_parent_hash FixedString(66) Codec(ZSTD(1)),
  meta_client_name LowCardinality(String),
  meta_client_id String Codec(ZSTD(1)),
  meta_client_version LowCardinality(String),
  meta_client_implementation LowCardinality(String),
  meta_client_os LowCardinality(String),
  meta_client_ip Nullable(IPv6) Codec(ZSTD(1)),
  meta_client_geo_city LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_country_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_continent_code LowCardinality(String) Codec(ZSTD(1)),
  meta_client_geo_longitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_latitude Nullable(Float64) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_number Nullable(UInt32) Codec(ZSTD(1)),
  meta_client_geo_autonomous_system_organization Nullable(String) Codec(ZSTD(1)),
  meta_network_id Int32 Codec(DoubleDelta, ZSTD(1)),
  meta_network_name LowCardinality(String),
  meta_execution_fork_id_hash LowCardinality(String),
  meta_execution_fork_id_next LowCardinality(String),
  meta_labels Map(String, String) Codec(ZSTD(1)),
  PROJECTION projection_event_date_time (SELECT * ORDER BY event_date_time),
  PROJECTION projection_slot (SELECT * ORDER BY slot),
  PROJECTION projection_epoch (SELECT * ORDER BY epoch),
  PROJECTION projection_block_root (SELECT * ORDER BY block_root),
  PROJECTION projection_parent_root (SELECT * ORDER BY parent_root),
  PROJECTION projection_state_root (SELECT * ORDER BY state_root),
  PROJECTION projection_eth1_data_block_hash (SELECT * ORDER BY eth1_data_block_hash),
  PROJECTION projection_execution_payload_block_hash (SELECT * ORDER BY execution_payload_block_hash),
  PROJECTION projection_execution_payload_block_number (SELECT * ORDER BY execution_payload_block_number),
  PROJECTION projection_execution_payload_state_root (SELECT * ORDER BY execution_payload_state_root),
  PROJECTION projection_execution_payload_parent_hash (SELECT * ORDER BY execution_payload_parent_hash)
) Engine = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY toStartOfMonth(slot_start_date_time)
ORDER BY (slot_start_date_time, meta_network_name, meta_client_name)
TTL slot_start_date_time TO VOLUME 'default',
    slot_start_date_time + INTERVAL 3 MONTH DELETE WHERE meta_network_name != 'mainnet',
    slot_start_date_time + INTERVAL 6 MONTH TO VOLUME 'hdd1',
    slot_start_date_time + INTERVAL 18 MONTH TO VOLUME 'hdd2',
    slot_start_date_time + INTERVAL 40 MONTH DELETE WHERE meta_network_name = 'mainnet';

CREATE TABLE beacon_api_eth_v2_beacon_block on cluster '{cluster}' AS beacon_api_eth_v2_beacon_block_local
ENGINE = Distributed('{cluster}', default, beacon_api_eth_v2_beacon_block_local, rand());
