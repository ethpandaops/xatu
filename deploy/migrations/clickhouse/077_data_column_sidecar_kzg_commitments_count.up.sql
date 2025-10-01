DROP TABLE IF EXISTS beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}' SYNC;

ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS kzg_commitments;

ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
    ADD COLUMN kzg_commitments_count UInt32 COMMENT 'Number of KZG commitments associated with the record' CODEC(ZSTD(1)) AFTER column_index;

CREATE TABLE default.beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}' AS default.beacon_api_eth_v1_events_data_column_sidecar_local ENGINE = Distributed(
    '{cluster}',
    default,
    beacon_api_eth_v1_events_data_column_sidecar_local,
    cityHash64(
        slot_start_date_time,
        meta_network_name,
        meta_client_name,
        block_root,
        column_index
    )
);