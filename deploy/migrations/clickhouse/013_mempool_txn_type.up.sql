ALTER TABLE mempool_transaction_local on cluster '{cluster}'
MODIFY COLUMN to Nullable(FixedString(42)) Codec(ZSTD(1));

ALTER TABLE mempool_transaction on cluster '{cluster}'
MODIFY COLUMN to Nullable(FixedString(42)) Codec(ZSTD(1));

ALTER TABLE mempool_transaction_local on cluster '{cluster}'
ADD COLUMN type Nullable(UInt8) Codec(ZSTD(1)) AFTER value;

ALTER TABLE mempool_transaction on cluster '{cluster}'
ADD COLUMN type Nullable(UInt8) Codec(ZSTD(1)) AFTER value;

ALTER TABLE mempool_transaction_local on cluster '{cluster}'
COMMENT COLUMN type 'The type of the transaction';

ALTER TABLE mempool_transaction on cluster '{cluster}'
COMMENT COLUMN type 'The type of the transaction';
