-- EIP-7732 ePBS gossip-table corrections.
--
-- Section 1: make propagation_slot_start_diff signed.
--   Builder bids and proposer preferences are broadcast BEFORE their slot begins,
--   so (event_time - slot_start) is negative. The column was UInt32, which stored
--   the negative value as a ~4.29e9 underflow. Change it to Int32 on the two
--   pre-slot event tables (SSE + gossip). The UInt32 -> Int32 conversion is a
--   wrapping cast in ClickHouse, so existing underflowed rows are corrected to
--   their true negative value in place (e.g. 4294966896 -> -400).
--
-- Section 2: drop the vestigial `version` column.
--   The four ePBS gossip tables carry a `version UInt32 DEFAULT
--   4294967295 - propagation_slot_start_diff` column commented "for dedup", but no
--   table's ReplacingMergeTree uses it -- every engine keys on updated_date_time.
--   The same column was already removed from the pre-Gloas gossip tables
--   (29466e46) and unintentionally reintroduced here. Drop it.

---------------------------------------------------------------------
-- Section 1: signed propagation_slot_start_diff (bid + proposer_preferences)
---------------------------------------------------------------------

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_proposer_preferences_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_proposer_preferences ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Difference between event_date_time and slot_start_date_time in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Propagation delay from slot start in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Propagation delay from slot start in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_proposer_preferences_local ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Propagation delay from slot start in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_proposer_preferences ON CLUSTER '{cluster}'
    MODIFY COLUMN propagation_slot_start_diff Int32
        COMMENT 'Propagation delay from slot start in ms (negative if observed before slot start)'
        CODEC(ZSTD(1));

---------------------------------------------------------------------
-- Section 2: drop vestigial `version` column (all four ePBS gossip tables)
---------------------------------------------------------------------

ALTER TABLE libp2p_gossipsub_execution_payload_envelope ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;
ALTER TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;

ALTER TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;
ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;

ALTER TABLE libp2p_gossipsub_payload_attestation_message ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;
ALTER TABLE libp2p_gossipsub_payload_attestation_message_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;

ALTER TABLE libp2p_gossipsub_proposer_preferences ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;
ALTER TABLE libp2p_gossipsub_proposer_preferences_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS version;
