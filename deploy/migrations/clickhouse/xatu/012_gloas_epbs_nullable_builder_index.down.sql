-- Reverse of 012.
--
-- The sentinel is restored before the type is narrowed, because a plain
-- Nullable(UInt64) -> UInt64 cast would silently turn NULL into 0 and make
-- self-built payloads indistinguishable from builder index 0.

---------------------------------------------------------------------
-- NULL back to the UInt64-max sentinel
---------------------------------------------------------------------

ALTER TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}'
    UPDATE builder_index = 18446744073709551615 WHERE builder_index IS NULL;

ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}'
    UPDATE builder_index = 18446744073709551615 WHERE builder_index IS NULL;

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}'
    UPDATE builder_index = 18446744073709551615 WHERE builder_index IS NULL;

ALTER TABLE canonical_beacon_block_execution_payload_bid_local ON CLUSTER '{cluster}'
    UPDATE builder_index = 18446744073709551615 WHERE builder_index IS NULL;

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    UPDATE builder_index = 18446744073709551615 WHERE builder_index IS NULL;

ALTER TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}'
    UPDATE builder_index = 18446744073709551615 WHERE builder_index IS NULL;

---------------------------------------------------------------------
-- builder_index back to UInt64
---------------------------------------------------------------------

ALTER TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder that produced the payload'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder that produced the payload'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder that produced the payload'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder that produced the payload'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder'
        CODEC(ZSTD(1));

ALTER TABLE canonical_beacon_block_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder in the builder registry'
        CODEC(ZSTD(1));
ALTER TABLE canonical_beacon_block_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder in the builder registry'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder'
        CODEC(ZSTD(1));
ALTER TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder that produced the payload'
        CODEC(ZSTD(1));
ALTER TABLE libp2p_gossipsub_execution_payload_envelope ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index UInt64
        COMMENT 'Index of the builder that produced the payload'
        CODEC(ZSTD(1));
