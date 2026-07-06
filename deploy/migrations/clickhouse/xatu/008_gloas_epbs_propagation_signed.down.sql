-- Reverse of 008: restore the vestigial `version` column and revert
-- propagation_slot_start_diff to UInt32.

---------------------------------------------------------------------
-- Section 2 reverse: re-add `version` column (all four ePBS gossip tables)
---------------------------------------------------------------------

ALTER TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;
ALTER TABLE libp2p_gossipsub_execution_payload_envelope ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;
ALTER TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;

ALTER TABLE libp2p_gossipsub_payload_attestation_message_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;
ALTER TABLE libp2p_gossipsub_payload_attestation_message ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;

ALTER TABLE libp2p_gossipsub_proposer_preferences_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;
ALTER TABLE libp2p_gossipsub_proposer_preferences ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS version UInt32 DEFAULT 4294967295 - propagation_slot_start_diff
        COMMENT 'Version for dedup: prefer lowest propagation time'
        CODEC(DoubleDelta, ZSTD(1)) AFTER updated_date_time;

---------------------------------------------------------------------
-- Section 1 reverse: propagation_slot_start_diff back to UInt32
---------------------------------------------------------------------

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_proposer_preferences_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_proposer_preferences ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Propagation delay from slot start in ms'
        CODEC(ZSTD(1));
ALTER TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Propagation delay from slot start in ms'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_proposer_preferences_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Propagation delay from slot start in ms'
        CODEC(ZSTD(1));
ALTER TABLE libp2p_gossipsub_proposer_preferences ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff UInt32
        COMMENT 'Propagation delay from slot start in ms'
        CODEC(ZSTD(1));
