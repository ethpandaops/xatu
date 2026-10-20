-- New columns are appended (no AFTER) so the generated pkg/proto/clickhouse
-- field numbers of existing columns stay stable.

-- canonical_beacon_elaborated_attestation: from Gloas the attestation data
-- index is the attester's payload vote (EIP-7732), 1 when the payload of the
-- voted block was seen. The committee index comes from committee_bits.
ALTER TABLE canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS payload_present Nullable(Bool)
        COMMENT 'Whether the attesters saw the execution payload of the voted block (attestation data index 1). Always false when the voted block is from the attestation''s own slot. Null before Gloas' CODEC(ZSTD(1));

ALTER TABLE canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS payload_present Nullable(Bool)
        COMMENT 'Whether the attesters saw the execution payload of the voted block (attestation data index 1). Always false when the voted block is from the attestation''s own slot. Null before Gloas' CODEC(ZSTD(1));

-- canonical_beacon_block_execution_transaction: EIP-7702 authorization lists,
-- one array element per authorization.
ALTER TABLE canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS authorization_chain_ids Array(UInt256)
        COMMENT 'The chain ID of each EIP-7702 authorization in a set code transaction' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS authorization_addresses Array(FixedString(42))
        COMMENT 'The address each EIP-7702 authorization delegates its authority''s code to' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS authorization_nonces Array(UInt64)
        COMMENT 'The authority nonce of each EIP-7702 authorization' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS authorization_authorities Array(String)
        COMMENT 'The account that signed each EIP-7702 authorization, recovered from its signature. Empty when the signature does not recover' CODEC(ZSTD(1));

ALTER TABLE canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS authorization_chain_ids Array(UInt256)
        COMMENT 'The chain ID of each EIP-7702 authorization in a set code transaction' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS authorization_addresses Array(FixedString(42))
        COMMENT 'The address each EIP-7702 authorization delegates its authority''s code to' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS authorization_nonces Array(UInt64)
        COMMENT 'The authority nonce of each EIP-7702 authorization' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS authorization_authorities Array(String)
        COMMENT 'The account that signed each EIP-7702 authorization, recovered from its signature. Empty when the signature does not recover' CODEC(ZSTD(1));

-- canonical_beacon_block: payload_present is now set from whether the
-- block's payload envelope was revealed.
ALTER TABLE canonical_beacon_block_local ON CLUSTER '{cluster}'
    COMMENT COLUMN payload_present 'Whether the builder revealed the execution payload (Gloas+)';

ALTER TABLE canonical_beacon_block ON CLUSTER '{cluster}'
    COMMENT COLUMN payload_present 'Whether the builder revealed the execution payload (Gloas+)';

-- canonical_execution_block: the remaining payload header fields that
-- canonical_beacon_block carries before Gloas, so every fork has them here.
ALTER TABLE canonical_execution_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS parent_hash Nullable(FixedString(66))
        COMMENT 'The parent block hash' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS state_root Nullable(FixedString(66))
        COMMENT 'The state root after executing the block' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS blob_gas_used Nullable(UInt64)
        COMMENT 'The blob gas used by the block (EIP-4844). Null before Cancun' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS excess_blob_gas Nullable(UInt64)
        COMMENT 'The excess blob gas of the block (EIP-4844). Null before Cancun' CODEC(ZSTD(1));

ALTER TABLE canonical_execution_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS parent_hash Nullable(FixedString(66))
        COMMENT 'The parent block hash' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS state_root Nullable(FixedString(66))
        COMMENT 'The state root after executing the block' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS blob_gas_used Nullable(UInt64)
        COMMENT 'The blob gas used by the block (EIP-4844). Null before Cancun' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS excess_blob_gas Nullable(UInt64)
        COMMENT 'The excess blob gas of the block (EIP-4844). Null before Cancun' CODEC(ZSTD(1));
