-- Transaction pool lifecycle: transactions that left a sentry's pool without being included, and per-peer blob
-- transaction announcements seen by mimicry.

CREATE TABLE IF NOT EXISTS mempool_transaction_removed_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the sentry found the transaction gone from its execution client transaction pool' CODEC(DoubleDelta, ZSTD(1)),
    `first_seen_date_time` DateTime64(3) COMMENT 'When the transaction was first in a txpool_content snapshot from this sentry' CODEC(DoubleDelta, ZSTD(1)),
    `last_seen_date_time` DateTime64(3) COMMENT 'When the transaction was last in a txpool_content snapshot from this sentry' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'The hash of the transaction' CODEC(ZSTD(1)),
    `from` FixedString(42) COMMENT 'The address of the account that sent the transaction' CODEC(ZSTD(1)),
    `to` Nullable(FixedString(42)) COMMENT 'The address of the transaction recipient, NULL for contract creations' CODEC(ZSTD(1)),
    `nonce` UInt64 COMMENT 'The nonce of the transaction' CODEC(ZSTD(1)),
    `type` UInt8 COMMENT 'The type of the transaction' CODEC(ZSTD(1)),
    `gas` UInt64 COMMENT 'The maximum gas provided for the transaction execution' CODEC(ZSTD(1)),
    `gas_tip_cap` UInt128 COMMENT 'The priority fee (tip) cap in wei, equal to the gas price for legacy and access list transactions' CODEC(ZSTD(1)),
    `gas_fee_cap` UInt128 COMMENT 'The max fee cap in wei, equal to the gas price for legacy and access list transactions' CODEC(ZSTD(1)),
    `blob_gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee per blob gas in wei, NULL for non-blob transactions' CODEC(ZSTD(1)),
    `reason` LowCardinality(String) COMMENT 'Why it left the pool: replaced (another transaction with the same sender and nonce is in the pool), nonce_consumed (the sender nonce moved past it on chain with another transaction) or dropped (neither, for example evicted or expired)',
    `pool` LowCardinality(String) COMMENT 'The txpool_content section that last held it: pending or queued',
    `replacement_hash` Nullable(FixedString(66)) COMMENT 'The hash of the transaction with the same sender and nonce that replaced it, set when the reason is replaced' CODEC(ZSTD(1)),
    `replacement_gas_tip_cap` Nullable(UInt128) COMMENT 'The priority fee (tip) cap in wei of the replacing transaction, set when the reason is replaced' CODEC(ZSTD(1)),
    `replacement_gas_fee_cap` Nullable(UInt128) COMMENT 'The max fee cap in wei of the replacing transaction, set when the reason is replaced' CODEC(ZSTD(1)),
    `account_nonce` Nullable(UInt64) COMMENT 'The sender nonce at the latest block when it was found gone, NULL when the reason is replaced' CODEC(ZSTD(1)),
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
PARTITION BY (meta_network_name, toYYYYMM(event_date_time))
ORDER BY (meta_network_name, event_date_time, meta_client_name, hash)
COMMENT 'Each row is a transaction that left a sentry execution client transaction pool without being included in a block, found by diffing successive txpool_content snapshots. Included transactions are not recorded.';

CREATE TABLE IF NOT EXISTS mempool_transaction_removed ON CLUSTER '{cluster}'
AS mempool_transaction_removed_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'mempool_transaction_removed_local', cityHash64(event_date_time, meta_network_name, meta_client_name, hash))
COMMENT 'Each row is a transaction that left a sentry execution client transaction pool without being included in a block, found by diffing successive txpool_content snapshots. Included transactions are not recorded.';

CREATE TABLE IF NOT EXISTS mempool_blob_transaction_announcement_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When mimicry received the announcement from the peer' CODEC(DoubleDelta, ZSTD(1)),
    `hash` FixedString(66) COMMENT 'The hash of the announced blob transaction' CODEC(ZSTD(1)),
    `size` UInt32 COMMENT 'The transaction size in bytes the peer announced, including blobs' CODEC(ZSTD(1)),
    `peer_implementation` LowCardinality(String) COMMENT 'Execution client implementation of the announcing peer, from its devp2p hello',
    `peer_version` LowCardinality(String) COMMENT 'Release version (for example 1.16.0) of the announcing peer, from its devp2p hello, empty when it has none',
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
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name',
    `meta_execution_fork_id_hash` LowCardinality(String) COMMENT 'The hash of the fork ID of the current Ethereum network',
    `meta_execution_fork_id_next` LowCardinality(String) COMMENT 'The fork ID of the next planned Ethereum network upgrade'
)
ENGINE = ReplicatedMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}')
PARTITION BY (meta_network_name, toYYYYMM(event_date_time))
ORDER BY (meta_network_name, event_date_time, meta_client_name, hash)
COMMENT 'Each row is one execution layer peer announcing a blob transaction hash to a mimicry client over devp2p. Peers are described only by client implementation and release version, not by node ID or address.';

CREATE TABLE IF NOT EXISTS mempool_blob_transaction_announcement ON CLUSTER '{cluster}'
AS mempool_blob_transaction_announcement_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'mempool_blob_transaction_announcement_local', cityHash64(event_date_time, meta_network_name, meta_client_name, hash))
COMMENT 'Each row is one execution layer peer announcing a blob transaction hash to a mimicry client over devp2p. Peers are described only by client implementation and release version, not by node ID or address.';
