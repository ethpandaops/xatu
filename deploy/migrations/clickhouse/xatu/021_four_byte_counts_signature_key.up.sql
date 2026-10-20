-- Rebuild canonical_execution_four_byte_counts_local with signature in its ORDER BY, keeping the old table as *_migrate_bak_local.

CREATE TABLE IF NOT EXISTS canonical_execution_four_byte_counts_migrate_bak_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `block_number` UInt64 COMMENT 'The block number' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_index` UInt64 COMMENT 'The transaction index in the block' CODEC(DoubleDelta, ZSTD(1)),
    `transaction_hash` FixedString(66) COMMENT 'The transaction hash' CODEC(ZSTD(1)),
    `signature` String COMMENT 'The signature of the four byte count' CODEC(ZSTD(1)),
    `size` UInt64 COMMENT 'The size of the four byte count' CODEC(ZSTD(1)),
    `count` UInt64 COMMENT 'The count of the four byte count' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/canonical_execution_four_byte_counts_v2_local', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, intDiv(block_number, 5000000))
ORDER BY (meta_network_name, block_number, transaction_hash, signature)
COMMENT 'Contains canonical execution four byte count data.';

CREATE TABLE IF NOT EXISTS canonical_execution_four_byte_counts_migrate_bak ON CLUSTER '{cluster}'
AS canonical_execution_four_byte_counts_migrate_bak_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'canonical_execution_four_byte_counts_migrate_bak_local', cityHash64(block_number, meta_network_name, transaction_hash));

-- Copy shard-locally into the new-key table while writers keep using the old one.
INSERT INTO canonical_execution_four_byte_counts_migrate_bak
    (updated_date_time, block_number, transaction_index, transaction_hash, signature, size, count, meta_network_name)
SELECT updated_date_time, block_number, transaction_index, transaction_hash, signature, size, count, meta_network_name
FROM canonical_execution_four_byte_counts
SETTINGS parallel_distributed_insert_select = 2, distributed_foreground_insert = 1, max_threads = 4, max_insert_threads = 4, min_insert_block_size_rows = 10000000, min_insert_block_size_bytes = 1073741824;

-- Not re-runnable past this point: a second EXCHANGE swaps the tables back.
EXCHANGE TABLES canonical_execution_four_byte_counts_local AND canonical_execution_four_byte_counts_migrate_bak_local ON CLUSTER '{cluster}';

-- Recopy recent rows so inserts that landed in the old table during the copy are kept.
INSERT INTO canonical_execution_four_byte_counts
    (updated_date_time, block_number, transaction_index, transaction_hash, signature, size, count, meta_network_name)
SELECT updated_date_time, block_number, transaction_index, transaction_hash, signature, size, count, meta_network_name
FROM canonical_execution_four_byte_counts_migrate_bak
WHERE updated_date_time >= now() - INTERVAL 1 DAY
SETTINGS parallel_distributed_insert_select = 2, distributed_foreground_insert = 1, max_threads = 4, max_insert_threads = 4, min_insert_block_size_rows = 10000000, min_insert_block_size_bytes = 1073741824;

DROP TABLE IF EXISTS canonical_execution_four_byte_counts_migrate_bak ON CLUSTER '{cluster}' SYNC;
