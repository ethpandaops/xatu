ALTER TABLE canonical_beacon_blob_sidecar_local on cluster '{cluster}'
ADD COLUMN versioned_hash FixedString(66) Codec(ZSTD(1)) AFTER block_parent_root;

ALTER TABLE canonical_beacon_blob_sidecar on cluster '{cluster}'
ADD COLUMN versioned_hash FixedString(66) Codec(ZSTD(1)) AFTER block_parent_root;

ALTER TABLE canonical_beacon_blob_sidecar_local on cluster '{cluster}'
COMMENT COLUMN versioned_hash 'The versioned hash in the beacon API event stream payload';

ALTER TABLE canonical_beacon_blob_sidecar on cluster '{cluster}'
COMMENT COLUMN versioned_hash 'The versioned hash in the beacon API event stream payload';
