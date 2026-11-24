-- kzg_commitments_count: 0 means unknown (a data column sidecar implies at least one blob).

ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record. 0 when the beacon node omits kzg_commitments (beacon-APIs #583)';
ALTER TABLE beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record. 0 when the beacon node omits kzg_commitments (beacon-APIs #583)';

ALTER TABLE libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record. 0 on Gloas, where the sidecar no longer carries the block header';
ALTER TABLE libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
    COMMENT COLUMN kzg_commitments_count 'Number of KZG commitments associated with the record. 0 on Gloas, where the sidecar no longer carries the block header';
