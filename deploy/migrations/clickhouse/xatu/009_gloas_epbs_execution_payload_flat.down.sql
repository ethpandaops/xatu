-- Revert 009: restore the envelope-summary columns and drop execution_optimistic.

---------------------------------------------------------------------
-- execution_payload
---------------------------------------------------------------------
ALTER TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS execution_optimistic,
    ADD COLUMN IF NOT EXISTS state_root FixedString(66)
        COMMENT 'Execution state root' CODEC(ZSTD(1)) AFTER block_hash,
    ADD COLUMN IF NOT EXISTS slot_number Nullable(UInt64)
        COMMENT 'EIP-7843 SLOTNUM: the execution payload slot_number field (typically equals the beacon slot)' CODEC(ZSTD(1)) AFTER state_root;

ALTER TABLE beacon_api_eth_v1_events_execution_payload ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS execution_optimistic,
    ADD COLUMN IF NOT EXISTS state_root FixedString(66)
        COMMENT 'Execution state root' CODEC(ZSTD(1)) AFTER block_hash,
    ADD COLUMN IF NOT EXISTS slot_number Nullable(UInt64)
        COMMENT 'EIP-7843 SLOTNUM: the execution payload slot_number field (typically equals the beacon slot)' CODEC(ZSTD(1)) AFTER state_root;

---------------------------------------------------------------------
-- execution_payload_gossip
---------------------------------------------------------------------
ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS state_root FixedString(66)
        COMMENT 'Execution state root' CODEC(ZSTD(1)) AFTER block_hash,
    ADD COLUMN IF NOT EXISTS slot_number Nullable(UInt64)
        COMMENT 'EIP-7843 SLOTNUM: the execution payload slot_number field (typically equals the beacon slot)' CODEC(ZSTD(1)) AFTER state_root;

ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS state_root FixedString(66)
        COMMENT 'Execution state root' CODEC(ZSTD(1)) AFTER block_hash,
    ADD COLUMN IF NOT EXISTS slot_number Nullable(UInt64)
        COMMENT 'EIP-7843 SLOTNUM: the execution payload slot_number field (typically equals the beacon slot)' CODEC(ZSTD(1)) AFTER state_root;
