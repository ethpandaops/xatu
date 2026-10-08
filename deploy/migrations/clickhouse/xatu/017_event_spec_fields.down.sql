ALTER TABLE beacon_api_eth_v1_events_fast_confirmation ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS current_slot;

ALTER TABLE beacon_api_eth_v1_events_fast_confirmation_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS current_slot;

ALTER TABLE beacon_api_eth_v1_events_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS block_hash,
    DROP COLUMN IF EXISTS builder_index;

ALTER TABLE beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS block_hash,
    DROP COLUMN IF EXISTS builder_index;
