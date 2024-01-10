ALTER TABLE mempool_transaction_local on cluster '{cluster}'
    DROP COLUMN type;

ALTER TABLE mempool_transaction on cluster '{cluster}'
    DROP COLUMN type;

ALTER TABLE mempool_transaction_local on cluster '{cluster}'
MODIFY COLUMN to FixedString(42) Codec(ZSTD(1));

ALTER TABLE mempool_transaction on cluster '{cluster}'
MODIFY COLUMN to FixedString(42) Codec(ZSTD(1));
