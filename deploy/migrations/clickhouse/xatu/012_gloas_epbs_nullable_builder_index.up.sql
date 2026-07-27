-- Gloas/ePBS builder_index UInt64 -> Nullable(UInt64).
--
-- Beacon nodes emit "builder_index": "18446744073709551615" (UInt64 max) to
-- mean "no builder" for a self-built payload, and ~24% of execution_payload
-- rows carry it. Stored as a magic number it silently destroys max()/avg()
-- over the column. NULL is already how builder_index is modelled on
-- canonical_beacon_block and beacon_api_eth_v2_beacon_block, so this aligns
-- the ePBS tables with it.
--
-- builder_index does not participate in any of these tables' ORDER BY, so the
-- type change and the sentinel rewrite are both in-place ALTERs.

---------------------------------------------------------------------
-- Type change
---------------------------------------------------------------------

ALTER TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder that produced the payload, NULL when self-built'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder that produced the payload, NULL when self-built'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder that produced the payload, NULL when self-built'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder that produced the payload, NULL when self-built'
        CODEC(ZSTD(1));

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder, NULL when self-built'
        CODEC(ZSTD(1));
ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder, NULL when self-built'
        CODEC(ZSTD(1));

ALTER TABLE canonical_beacon_block_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder in the builder registry, NULL when self-built'
        CODEC(ZSTD(1));
ALTER TABLE canonical_beacon_block_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder in the builder registry, NULL when self-built'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder, NULL when self-built'
        CODEC(ZSTD(1));
ALTER TABLE libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder, NULL when self-built'
        CODEC(ZSTD(1));

ALTER TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder that produced the payload, NULL when self-built'
        CODEC(ZSTD(1));
ALTER TABLE libp2p_gossipsub_execution_payload_envelope ON CLUSTER '{cluster}'
    MODIFY COLUMN builder_index Nullable(UInt64)
        COMMENT 'Index of the builder that produced the payload, NULL when self-built'
        CODEC(ZSTD(1));

---------------------------------------------------------------------
-- Rewrite the historical UInt64-max sentinel to NULL.
--
-- Mutations run against the _local tables only, as the Distributed tables have
-- no data of their own. Each is scoped by the sentinel so the mutation only
-- rewrites the affected parts.
---------------------------------------------------------------------

ALTER TABLE beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}'
    UPDATE builder_index = NULL WHERE builder_index = 18446744073709551615;

ALTER TABLE beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}'
    UPDATE builder_index = NULL WHERE builder_index = 18446744073709551615;

ALTER TABLE beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}'
    UPDATE builder_index = NULL WHERE builder_index = 18446744073709551615;

ALTER TABLE canonical_beacon_block_execution_payload_bid_local ON CLUSTER '{cluster}'
    UPDATE builder_index = NULL WHERE builder_index = 18446744073709551615;

ALTER TABLE libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}'
    UPDATE builder_index = NULL WHERE builder_index = 18446744073709551615;

ALTER TABLE libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}'
    UPDATE builder_index = NULL WHERE builder_index = 18446744073709551615;
