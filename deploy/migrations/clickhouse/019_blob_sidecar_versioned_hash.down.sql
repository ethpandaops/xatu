ALTER TABLE canonical_beacon_blob_sidecar_local on cluster '{cluster}'
    DROP COLUMN versioned_hash;

ALTER TABLE canonical_beacon_blob_sidecar on cluster '{cluster}'
    DROP COLUMN versioned_hash;
