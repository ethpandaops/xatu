ALTER TABLE default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}'
DROP COLUMN beacon_block_root;

ALTER TABLE default.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}'
DROP COLUMN beacon_block_root;

ALTER TABLE default.libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
DROP COLUMN beacon_block_root;

ALTER TABLE default.libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
DROP COLUMN beacon_block_root;
