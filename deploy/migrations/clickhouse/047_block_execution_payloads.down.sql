ALTER TABLE canonical_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN execution_payload_base_fee_per_gas,
    DROP COLUMN execution_payload_blob_gas_used,
    DROP COLUMN execution_payload_excess_blob_gas,
    DROP COLUMN execution_payload_gas_limit,
    DROP COLUMN execution_payload_gas_used;

ALTER TABLE canonical_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN execution_payload_base_fee_per_gas,
    DROP COLUMN execution_payload_blob_gas_used,
    DROP COLUMN execution_payload_excess_blob_gas,
    DROP COLUMN execution_payload_gas_limit,
    DROP COLUMN execution_payload_gas_used;

ALTER TABLE beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN execution_payload_base_fee_per_gas,
    DROP COLUMN execution_payload_blob_gas_used,
    DROP COLUMN execution_payload_excess_blob_gas,
    DROP COLUMN execution_payload_gas_limit,
    DROP COLUMN execution_payload_gas_used;

ALTER TABLE beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN execution_payload_base_fee_per_gas,
    DROP COLUMN execution_payload_blob_gas_used,
    DROP COLUMN execution_payload_excess_blob_gas,
    DROP COLUMN execution_payload_gas_limit,
    DROP COLUMN execution_payload_gas_used;
