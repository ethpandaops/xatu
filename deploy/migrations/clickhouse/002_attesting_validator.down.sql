ALTER TABLE beacon_api_eth_v1_events_attestation on cluster '{cluster}'
    DROP COLUMN attesting_validator_index,
    DROP COLUMN attesting_validator_committee_index;

ALTER TABLE beacon_api_eth_v1_events_attestation_local on cluster '{cluster}'
    DROP COLUMN attesting_validator_index,
    DROP COLUMN attesting_validator_committee_index;
