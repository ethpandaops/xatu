CREATE DATABASE IF NOT EXISTS mainnet ON CLUSTER '{cluster}';

CREATE VIEW mainnet.canonical_execution_nonce_reads ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_nonce_reads
WHERE
    meta_network_name = 'mainnet';

CREATE VIEW mainnet.canonical_execution_nonce_diffs ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_nonce_diffs
WHERE
    meta_network_name = 'mainnet';

CREATE VIEW mainnet.canonical_execution_balance_diffs ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_balance_diffs
WHERE
    meta_network_name = 'mainnet';

CREATE VIEW mainnet.canonical_execution_balance_reads ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_balance_reads
WHERE
    meta_network_name = 'mainnet';

CREATE VIEW mainnet.canonical_execution_storage_diffs ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_storage_diffs
WHERE
    meta_network_name = 'mainnet';

CREATE VIEW mainnet.canonical_execution_storage_reads ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_storage_reads
WHERE
    meta_network_name = 'mainnet';

CREATE VIEW mainnet.canonical_execution_contracts ON CLUSTER '{cluster}' AS
SELECT
    *
FROM
    default.canonical_execution_contracts
WHERE
    meta_network_name = 'mainnet';

CREATE TABLE mainnet.stg_account_last_access_local on cluster '{cluster}' (
    `address` String COMMENT 'The address of the account' CODEC(ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number of the last access' CODEC(ZSTD(1)),
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    `block_number`
) PARTITION BY cityHash64(`address`) % 16
ORDER BY
    (address) COMMENT 'Table for accounts last access data';

CREATE TABLE mainnet.stg_account_last_access ON CLUSTER '{cluster}' AS mainnet.stg_account_last_access_local ENGINE = Distributed(
    '{cluster}',
    mainnet,
    stg_account_last_access_local,
    cityHash64(`address`)
);

CREATE TABLE mainnet.stg_account_first_access_local on cluster '{cluster}' (
    `address` String COMMENT 'The address of the account' CODEC(ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number of the first access' CODEC(ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - block_number COMMENT 'Version for this address, for internal use in clickhouse to keep first access' CODEC(DoubleDelta, ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    `version`
) PARTITION BY cityHash64(`address`) % 16
ORDER BY
    (address) COMMENT 'Table for accounts first access data';

CREATE TABLE mainnet.stg_account_first_access ON CLUSTER '{cluster}' AS mainnet.stg_account_first_access_local ENGINE = Distributed(
    '{cluster}',
    mainnet,
    stg_account_first_access_local,
    cityHash64(`address`)
);

CREATE TABLE mainnet.stg_storage_last_access_local on cluster '{cluster}' (
    `address` String COMMENT 'The address of the account' CODEC(ZSTD(1)),
    `slot_key` String COMMENT 'The slot key of the storage' CODEC(ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number of the last access' CODEC(ZSTD(1)),
    `value` String COMMENT 'The value of the storage' CODEC(ZSTD(1)),
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    `block_number`
) PARTITION BY cityHash64(`address`) % 16
ORDER BY (address, slot_key) COMMENT 'Table for storage last access data';

CREATE TABLE mainnet.stg_storage_last_access ON CLUSTER '{cluster}' AS mainnet.stg_storage_last_access_local ENGINE = Distributed(
    '{cluster}',
    mainnet,
    stg_storage_last_access_local,
    cityHash64(`address`, `slot_key`)
);

CREATE TABLE mainnet.stg_storage_first_access_local on cluster '{cluster}' (
    `address` String COMMENT 'The address of the account' CODEC(ZSTD(1)),
    `slot_key` String COMMENT 'The slot key of the storage' CODEC(ZSTD(1)),
    `block_number` UInt32 COMMENT 'The block number of the first access' CODEC(ZSTD(1)),
    `value` String COMMENT 'The value of the storage' CODEC(ZSTD(1)),
    `version` UInt32 DEFAULT 4294967295 - block_number COMMENT 'Version for this address + slot key, for internal use in clickhouse to keep first access' CODEC(DoubleDelta, ZSTD(1))
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}',
    '{replica}',
    `version`
) PARTITION BY cityHash64(`address`) % 16
ORDER BY (address, slot_key) COMMENT 'Table for storage first access data';

CREATE TABLE mainnet.stg_storage_first_access ON CLUSTER '{cluster}' AS mainnet.stg_storage_first_access_local ENGINE = Distributed(
    '{cluster}',
    mainnet,
    stg_storage_first_access_local,
    cityHash64(`address`, `slot_key`)
);
