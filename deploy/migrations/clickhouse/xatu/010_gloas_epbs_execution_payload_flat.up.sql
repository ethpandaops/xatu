-- EIP-7732 ePBS execution_payload / execution_payload_gossip event reshape.
--
-- The beacon-APIs execution_payload and execution_payload_gossip SSE events
-- carry a FLAT summary of a revealed payload, not a full
-- SignedExecutionPayloadEnvelope (see apis/eventstream/index.yaml):
--
--   execution_payload:        {slot, builder_index, block_hash, block_root, execution_optimistic}
--   execution_payload_gossip: {slot, builder_index, block_hash, block_root}
--
-- These tables were originally modelled by extracting a summary from the full
-- envelope, and so carry two columns the flat event cannot populate:
--   * state_root  (not in the event -- only nimbus emits an extra all-zero one)
--   * slot_number (not in the event, and the event slot is already stored)
-- Drop both. Add execution_optimistic to the execution_payload table (the gossip
-- event does not carry it, so its table is left without the column).
--
-- The tables are empty (the events never parsed), so these ALTERs touch no data.

---------------------------------------------------------------------
-- execution_payload (SSE, fires on import into fork-choice)
---------------------------------------------------------------------
ALTER TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS state_root,
    DROP COLUMN IF EXISTS slot_number,
    ADD COLUMN IF NOT EXISTS execution_optimistic Bool
        COMMENT 'Whether the node considered the payload optimistically imported' CODEC(ZSTD(1)) AFTER block_hash;

ALTER TABLE beacon_api_eth_v1_events_execution_payload ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS state_root,
    DROP COLUMN IF EXISTS slot_number,
    ADD COLUMN IF NOT EXISTS execution_optimistic Bool
        COMMENT 'Whether the node considered the payload optimistically imported' CODEC(ZSTD(1)) AFTER block_hash;

---------------------------------------------------------------------
-- execution_payload_gossip (SSE, fires on gossip validation)
---------------------------------------------------------------------
ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS state_root,
    DROP COLUMN IF EXISTS slot_number;

ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS state_root,
    DROP COLUMN IF EXISTS slot_number;
