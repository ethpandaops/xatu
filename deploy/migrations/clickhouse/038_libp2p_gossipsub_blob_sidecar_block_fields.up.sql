ALTER TABLE libp2p_gossipsub_blob_sidecar_local on cluster '{cluster}'
ADD COLUMN parent_root FixedString(66) Codec(ZSTD(1)) AFTER blob_index;

ALTER TABLE libp2p_gossipsub_blob_sidecar on cluster '{cluster}'
ADD COLUMN parent_root FixedString(66) Codec(ZSTD(1)) AFTER blob_index;

ALTER TABLE libp2p_gossipsub_blob_sidecar_local on cluster '{cluster}'
COMMENT COLUMN parent_root 'Parent root of the beacon block';

ALTER TABLE libp2p_gossipsub_blob_sidecar on cluster '{cluster}'
COMMENT COLUMN parent_root 'Parent root of the beacon block';

ALTER TABLE libp2p_gossipsub_blob_sidecar_local on cluster '{cluster}'
ADD COLUMN state_root FixedString(66) Codec(ZSTD(1)) AFTER parent_root;

ALTER TABLE libp2p_gossipsub_blob_sidecar on cluster '{cluster}'
ADD COLUMN state_root FixedString(66) Codec(ZSTD(1)) AFTER parent_root;

ALTER TABLE libp2p_gossipsub_blob_sidecar_local on cluster '{cluster}'
COMMENT COLUMN state_root 'State root of the beacon block';

ALTER TABLE libp2p_gossipsub_blob_sidecar on cluster '{cluster}'
COMMENT COLUMN state_root 'State root of the beacon block';
