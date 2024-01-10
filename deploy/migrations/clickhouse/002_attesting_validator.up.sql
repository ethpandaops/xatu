ALTER TABLE beacon_api_eth_v1_events_attestation_local on cluster '{cluster}'
    ADD COLUMN attesting_validator_index Nullable(UInt32) Codec(ZSTD(1)) AFTER committee_index,
    ADD COLUMN attesting_validator_committee_index LowCardinality(String) AFTER attesting_validator_index;

ALTER TABLE beacon_api_eth_v1_events_attestation on cluster '{cluster}'
    ADD COLUMN attesting_validator_index Nullable(UInt32) Codec(ZSTD(1)) AFTER committee_index,
    ADD COLUMN attesting_validator_committee_index LowCardinality(String) AFTER attesting_validator_index;
