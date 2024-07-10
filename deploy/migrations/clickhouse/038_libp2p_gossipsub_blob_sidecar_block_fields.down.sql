ALTER TABLE libp2p_gossipsub_blob_sidecar_local on cluster '{cluster}'
    DROP COLUMN parent_root;

ALTER TABLE libp2p_gossipsub_blob_sidecar on cluster '{cluster}'
    DROP COLUMN parent_root;

ALTER TABLE libp2p_gossipsub_blob_sidecar_local on cluster '{cluster}'
    DROP COLUMN state_root;

ALTER TABLE libp2p_gossipsub_blob_sidecar on cluster '{cluster}'
    DROP COLUMN state_root;
