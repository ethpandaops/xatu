-- Drop Gloas beacon state snapshot tables.

-- execution_payload_availability
DROP TABLE IF EXISTS canonical_beacon_state_execution_payload_availability ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_execution_payload_availability_local ON CLUSTER '{cluster}' SYNC;

-- builder_pending_withdrawal
DROP TABLE IF EXISTS canonical_beacon_state_builder_pending_withdrawal ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_builder_pending_withdrawal_local ON CLUSTER '{cluster}' SYNC;

-- builder_pending_payment
DROP TABLE IF EXISTS canonical_beacon_state_builder_pending_payment ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_builder_pending_payment_local ON CLUSTER '{cluster}' SYNC;

-- builder
DROP TABLE IF EXISTS canonical_beacon_state_builder ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_builder_local ON CLUSTER '{cluster}' SYNC;

-- ptc_member
DROP TABLE IF EXISTS canonical_beacon_state_ptc_member ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS canonical_beacon_state_ptc_member_local ON CLUSTER '{cluster}' SYNC;
