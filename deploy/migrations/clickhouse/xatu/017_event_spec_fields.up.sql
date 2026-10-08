-- New columns are appended (no AFTER) so the generated pkg/proto/clickhouse
-- field numbers of existing columns stay stable.

-- block event: builder_index and block_hash from the block's
-- signed_execution_payload_bid.message (beacon-APIs #641, Gloas onwards).
ALTER TABLE beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS builder_index Nullable(UInt64)
        COMMENT 'Index of the builder whose execution payload bid the block commits to. Null before Gloas or when the beacon node does not send it' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS block_hash Nullable(FixedString(66))
        COMMENT 'Execution block hash from the block''s execution payload bid. Null before Gloas or when the beacon node does not send it' CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS builder_index Nullable(UInt64)
        COMMENT 'Index of the builder whose execution payload bid the block commits to. Null before Gloas or when the beacon node does not send it' CODEC(ZSTD(1)),
    ADD COLUMN IF NOT EXISTS block_hash Nullable(FixedString(66))
        COMMENT 'Execution block hash from the block''s execution payload bid. Null before Gloas or when the beacon node does not send it' CODEC(ZSTD(1));

-- fast_confirmation event: current_slot (beacon-APIs #616).
ALTER TABLE beacon_api_eth_v1_events_fast_confirmation_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS current_slot Nullable(UInt32)
        COMMENT 'Wall-clock slot at which the beacon node ran the fast confirmation algorithm that first confirmed this block. Null when the beacon node does not send it' CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_fast_confirmation ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS current_slot Nullable(UInt32)
        COMMENT 'Wall-clock slot at which the beacon node ran the fast confirmation algorithm that first confirmed this block. Null when the beacon node does not send it' CODEC(ZSTD(1));
