-- Reverse EIP-7732 ePBS support

-- Drop new tables
DROP TABLE IF EXISTS canonical_beacon_block_payload_attestation ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS canonical_beacon_block_payload_attestation_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS canonical_beacon_block_execution_payload_bid ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS canonical_beacon_block_execution_payload_bid_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_api_eth_v1_events_payload_attestation ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_v1_events_payload_attestation_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_bid ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_bid_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_api_eth_v1_events_proposer_preferences ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_v1_events_proposer_preferences_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_gossip ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_gossip_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_available ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_api_eth_v1_events_execution_payload_available_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS libp2p_gossipsub_execution_payload_envelope ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS libp2p_gossipsub_execution_payload_envelope_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS libp2p_gossipsub_execution_payload_bid ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS libp2p_gossipsub_execution_payload_bid_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS libp2p_gossipsub_payload_attestation_message ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS libp2p_gossipsub_payload_attestation_message_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS libp2p_gossipsub_proposer_preferences ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS libp2p_gossipsub_proposer_preferences_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_synthetic_payload_status_resolved ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_synthetic_payload_status_resolved_local ON CLUSTER '{cluster}';

DROP TABLE IF EXISTS beacon_synthetic_builder_pending_payment_settlement ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS beacon_synthetic_builder_pending_payment_settlement_local ON CLUSTER '{cluster}';

-- Remove ePBS columns from beacon block tables
ALTER TABLE canonical_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS payload_present,
    DROP COLUMN IF EXISTS execution_payment,
    DROP COLUMN IF EXISTS bid_value,
    DROP COLUMN IF EXISTS builder_index;

ALTER TABLE canonical_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS payload_present,
    DROP COLUMN IF EXISTS execution_payment,
    DROP COLUMN IF EXISTS bid_value,
    DROP COLUMN IF EXISTS builder_index;

ALTER TABLE beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS payload_present,
    DROP COLUMN IF EXISTS execution_payment,
    DROP COLUMN IF EXISTS bid_value,
    DROP COLUMN IF EXISTS builder_index;

ALTER TABLE beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS payload_present,
    DROP COLUMN IF EXISTS execution_payment,
    DROP COLUMN IF EXISTS bid_value,
    DROP COLUMN IF EXISTS builder_index;

-- Remove withdrawal_type column added by 107
ALTER TABLE canonical_beacon_block_withdrawal ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS withdrawal_type;

ALTER TABLE canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS withdrawal_type;

