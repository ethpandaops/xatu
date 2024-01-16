ALTER TABLE default.beacon_p2p_attestation_local ON CLUSTER '{cluster}'
ADD COLUMN signature String Codec(ZSTD(1)) AFTER attesting_validator_committee_index;

ALTER TABLE default.beacon_p2p_attestation ON CLUSTER '{cluster}'
ADD COLUMN signature String Codec(ZSTD(1)) AFTER attesting_validator_committee_index;
