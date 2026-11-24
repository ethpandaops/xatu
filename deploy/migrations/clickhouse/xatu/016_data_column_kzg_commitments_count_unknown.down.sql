ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record';
ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record';

ALTER TABLE libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record';
ALTER TABLE libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record';
