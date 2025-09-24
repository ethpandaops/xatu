
ALTER TABLE default.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}'
ADD COLUMN beacon_block_root FixedString(66) Codec(ZSTD(1)) AFTER blob_index;

ALTER TABLE default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}'
ADD COLUMN beacon_block_root FixedString(66) Codec(ZSTD(1)) AFTER blob_index;

ALTER TABLE default.libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
ADD COLUMN beacon_block_root FixedString(66) Codec(ZSTD(1)) AFTER kzg_commitments_count;

ALTER TABLE default.libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
ADD COLUMN beacon_block_root FixedString(66) Codec(ZSTD(1)) AFTER kzg_commitments_count;
