ALTER TABLE default.beacon_p2p_attestation_local ON CLUSTER '{cluster}'
DROP COLUMN signature;

ALTER TABLE default.beacon_p2p_attestation ON CLUSTER '{cluster}'
DROP COLUMN signature;

ALTER TABLE default.beacon_api_eth_v1_events_attestation_local ON CLUSTER '{cluster}'
DROP COLUMN signature;

ALTER TABLE default.beacon_api_eth_v1_events_attestation ON CLUSTER '{cluster}'
DROP COLUMN signature;
