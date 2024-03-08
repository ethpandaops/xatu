ALTER TABLE mempool_transaction_local on cluster '{cluster}'
ADD COLUMN gas_tip_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas,
ADD COLUMN gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas_tip_cap,
ADD COLUMN blob_gas Nullable(UInt64) Codec(ZSTD(1)) AFTER call_data_size,
ADD COLUMN blob_gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER blob_gas,
ADD COLUMN blob_hashes Array(String) Codec(ZSTD(1)) AFTER blob_gas_fee_cap,
ADD COLUMN blob_sidecars_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_hashes,
ADD COLUMN blob_sidecars_empty_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_sidecars_size,
COMMENT COLUMN gas_tip_cap 'The priority fee (tip) the user has set for the transaction',
COMMENT COLUMN gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_gas 'The maximum gas provided for the blob transaction execution',
COMMENT COLUMN blob_gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_hashes 'The hashes of the blob commitments for blob transactions',
COMMENT COLUMN blob_sidecars_size 'The total size of the sidecars for blob transactions in bytes',
COMMENT COLUMN blob_sidecars_empty_size 'The total empty size of the sidecars for blob transactions in bytes';

ALTER TABLE mempool_transaction on cluster '{cluster}'
ADD COLUMN gas_tip_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas,
ADD COLUMN gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas_tip_cap,
ADD COLUMN blob_gas Nullable(UInt64) Codec(ZSTD(1)) AFTER call_data_size,
ADD COLUMN blob_gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER blob_gas,
ADD COLUMN blob_hashes Array(String) Codec(ZSTD(1)) AFTER blob_gas_fee_cap,
ADD COLUMN blob_sidecars_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_hashes,
ADD COLUMN blob_sidecars_empty_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_sidecars_size,
COMMENT COLUMN gas_tip_cap 'The priority fee (tip) the user has set for the transaction',
COMMENT COLUMN gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_gas 'The maximum gas provided for the blob transaction execution',
COMMENT COLUMN blob_gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_hashes 'The hashes of the blob commitments for blob transactions',
COMMENT COLUMN blob_sidecars_size 'The total size of the sidecars for blob transactions in bytes',
COMMENT COLUMN blob_sidecars_empty_size 'The total empty size of the sidecars for blob transactions in bytes';

ALTER TABLE canonical_beacon_block_execution_transaction_local on cluster '{cluster}'
ADD COLUMN gas_tip_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas,
ADD COLUMN gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas_tip_cap,
ADD COLUMN blob_gas Nullable(UInt64) Codec(ZSTD(1)) AFTER call_data_size,
ADD COLUMN blob_gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER blob_gas,
ADD COLUMN blob_hashes Array(String) Codec(ZSTD(1)) AFTER blob_gas_fee_cap,
ADD COLUMN blob_sidecars_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_hashes,
ADD COLUMN blob_sidecars_empty_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_sidecars_size,
COMMENT COLUMN gas_tip_cap 'The priority fee (tip) the user has set for the transaction',
COMMENT COLUMN gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_gas 'The maximum gas provided for the blob transaction execution',
COMMENT COLUMN blob_gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_hashes 'The hashes of the blob commitments for blob transactions',
COMMENT COLUMN blob_sidecars_size 'The total size of the sidecars for blob transactions in bytes',
COMMENT COLUMN blob_sidecars_empty_size 'The total empty size of the sidecars for blob transactions in bytes';

ALTER TABLE canonical_beacon_block_execution_transaction on cluster '{cluster}'
ADD COLUMN gas_tip_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas,
ADD COLUMN gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER gas_tip_cap,
ADD COLUMN blob_gas Nullable(UInt64) Codec(ZSTD(1)) AFTER call_data_size,
ADD COLUMN blob_gas_fee_cap Nullable(UInt128) Codec(ZSTD(1)) AFTER blob_gas,
ADD COLUMN blob_hashes Array(String) Codec(ZSTD(1)) AFTER blob_gas_fee_cap,
ADD COLUMN blob_sidecars_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_hashes,
ADD COLUMN blob_sidecars_empty_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_sidecars_size,
COMMENT COLUMN gas_tip_cap 'The priority fee (tip) the user has set for the transaction',
COMMENT COLUMN gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_gas 'The maximum gas provided for the blob transaction execution',
COMMENT COLUMN blob_gas_fee_cap 'The max fee the user has set for the transaction',
COMMENT COLUMN blob_hashes 'The hashes of the blob commitments for blob transactions',
COMMENT COLUMN blob_sidecars_size 'The total size of the sidecars for blob transactions in bytes',
COMMENT COLUMN blob_sidecars_empty_size 'The total empty size of the sidecars for blob transactions in bytes';

ALTER TABLE canonical_beacon_blob_sidecar_local on cluster '{cluster}'
ADD COLUMN blob_empty_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_size,
COMMENT COLUMN blob_empty_size 'The total empty size of the blob in bytes';

ALTER TABLE canonical_beacon_blob_sidecar on cluster '{cluster}'
ADD COLUMN blob_empty_size Nullable(UInt32) Codec(ZSTD(1)) AFTER blob_size,
COMMENT COLUMN blob_empty_size 'The total empty size of the blob in bytes';
