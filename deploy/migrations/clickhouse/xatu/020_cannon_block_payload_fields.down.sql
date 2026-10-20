ALTER TABLE canonical_beacon_block ON CLUSTER '{cluster}'
    COMMENT COLUMN execution_payload_slot_number '',
    COMMENT COLUMN execution_payload_block_access_list_root '',
    COMMENT COLUMN payload_present 'Whether execution payload was delivered (Gloas+)';

ALTER TABLE canonical_beacon_block_local ON CLUSTER '{cluster}'
    COMMENT COLUMN execution_payload_slot_number '',
    COMMENT COLUMN execution_payload_block_access_list_root '',
    COMMENT COLUMN payload_present 'Whether execution payload was delivered (Gloas+)';

ALTER TABLE canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS authorization_authorities,
    DROP COLUMN IF EXISTS authorization_nonces,
    DROP COLUMN IF EXISTS authorization_addresses,
    DROP COLUMN IF EXISTS authorization_chain_ids;

ALTER TABLE canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS authorization_authorities,
    DROP COLUMN IF EXISTS authorization_nonces,
    DROP COLUMN IF EXISTS authorization_addresses,
    DROP COLUMN IF EXISTS authorization_chain_ids;

ALTER TABLE canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS payload_present;

ALTER TABLE canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS payload_present;
