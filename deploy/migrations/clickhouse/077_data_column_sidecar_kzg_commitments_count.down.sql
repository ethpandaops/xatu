DROP TABLE IF EXISTS beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}';

ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS kzg_commitments_count;

ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
    ADD COLUMN kzg_commitments Array(FixedString(98)) COMMENT 'The KZG commitments in the beacon API event stream payload' CODEC(ZSTD(1)) AFTER column_index;

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