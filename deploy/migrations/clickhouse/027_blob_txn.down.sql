ALTER TABLE mempool_transaction_local on cluster '{cluster}'
    DROP COLUMN gas_tip_cap,
    DROP COLUMN gas_fee_cap,
    DROP COLUMN blob_gas,
    DROP COLUMN blob_gas_fee_cap,
    DROP COLUMN blob_hashes,
    DROP COLUMN blob_sidecars_size,
    DROP COLUMN blob_sidecars_empty_size;

ALTER TABLE mempool_transaction on cluster '{cluster}'
    DROP COLUMN gas_tip_cap,
    DROP COLUMN gas_fee_cap,
    DROP COLUMN blob_gas,
    DROP COLUMN blob_gas_fee_cap,
    DROP COLUMN blob_hashes,
    DROP COLUMN blob_sidecars_size,
    DROP COLUMN blob_sidecars_empty_size;

ALTER TABLE canonical_beacon_block_execution_transaction_local on cluster '{cluster}'
    DROP COLUMN gas_tip_cap,
    DROP COLUMN gas_fee_cap,
    DROP COLUMN blob_gas,
    DROP COLUMN blob_gas_fee_cap,
    DROP COLUMN blob_hashes,
    DROP COLUMN blob_sidecars_size,
    DROP COLUMN blob_sidecars_empty_size;

ALTER TABLE canonical_beacon_block_execution_transaction on cluster '{cluster}'
    DROP COLUMN gas_tip_cap,
    DROP COLUMN gas_fee_cap,
    DROP COLUMN blob_gas,
    DROP COLUMN blob_gas_fee_cap,
    DROP COLUMN blob_hashes,
    DROP COLUMN blob_sidecars_size,
    DROP COLUMN blob_sidecars_empty_size;

ALTER TABLE canonical_beacon_blob_sidecar_local on cluster '{cluster}'
    DROP COLUMN blob_empty_size;

ALTER TABLE canonical_beacon_blob_sidecar on cluster '{cluster}'
    DROP COLUMN blob_empty_size;
