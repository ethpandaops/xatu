-- Swap the old-key table back in, carry over every row written since, and drop the rebuilt table.

CREATE TABLE IF NOT EXISTS canonical_execution_four_byte_counts_migrate_bak ON CLUSTER '{cluster}'
AS canonical_execution_four_byte_counts_migrate_bak_local
ENGINE = Distributed('{cluster}', currentDatabase(), 'canonical_execution_four_byte_counts_migrate_bak_local', cityHash64(block_number, meta_network_name, transaction_hash));

EXCHANGE TABLES canonical_execution_four_byte_counts_local AND canonical_execution_four_byte_counts_migrate_bak_local ON CLUSTER '{cluster}';

INSERT INTO canonical_execution_four_byte_counts
    (updated_date_time, block_number, transaction_index, transaction_hash, signature, size, count, meta_network_name)
SELECT updated_date_time, block_number, transaction_index, transaction_hash, signature, size, count, meta_network_name
FROM canonical_execution_four_byte_counts_migrate_bak
SETTINGS parallel_distributed_insert_select = 2, distributed_foreground_insert = 1, max_threads = 4, max_insert_threads = 4, min_insert_block_size_rows = 10000000, min_insert_block_size_bytes = 1073741824;

DROP TABLE IF EXISTS canonical_execution_four_byte_counts_migrate_bak ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_execution_four_byte_counts_migrate_bak_local ON CLUSTER '{cluster}' SYNC;
