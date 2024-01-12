ALTER TABLE canonical_beacon_elaborated_attestation_local on cluster '{cluster}'
    DROP COLUMN inclusion_distance;

ALTER TABLE canonical_beacon_elaborated_attestation on cluster '{cluster}'
    DROP COLUMN inclusion_distance;
