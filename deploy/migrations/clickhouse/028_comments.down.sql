ALTER TABLE default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}'
MODIFY COMMENT '',
COMMENT COLUMN event_date_time '',
COMMENT COLUMN slot '',
COMMENT COLUMN slot_start_date_time '',
COMMENT COLUMN committee_index '',
COMMENT COLUMN validators '',
COMMENT COLUMN epoch '',
COMMENT COLUMN epoch_start_date_time '',
COMMENT COLUMN meta_client_name '',
COMMENT COLUMN meta_client_id '',
COMMENT COLUMN meta_client_version '',
COMMENT COLUMN meta_client_implementation '',
COMMENT COLUMN meta_client_os '',
COMMENT COLUMN meta_client_ip '',
COMMENT COLUMN meta_client_geo_city '',
COMMENT COLUMN meta_client_geo_country '',
COMMENT COLUMN meta_client_geo_country_code '',
COMMENT COLUMN meta_client_geo_continent_code '',
COMMENT COLUMN meta_client_geo_longitude '',
COMMENT COLUMN meta_client_geo_latitude '',
COMMENT COLUMN meta_client_geo_autonomous_system_number '',
COMMENT COLUMN meta_client_geo_autonomous_system_organization '',
COMMENT COLUMN meta_network_id '',
COMMENT COLUMN meta_network_name '',
COMMENT COLUMN meta_consensus_version '',
COMMENT COLUMN meta_consensus_version_major '',
COMMENT COLUMN meta_consensus_version_minor '',
COMMENT COLUMN meta_consensus_version_patch '',
COMMENT COLUMN meta_consensus_implementation '',
COMMENT COLUMN meta_labels '';

ALTER TABLE default.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT '',
COMMENT COLUMN event_date_time '',
COMMENT COLUMN slot '',
COMMENT COLUMN slot_start_date_time '',
COMMENT COLUMN committee_index '',
COMMENT COLUMN validators '',
COMMENT COLUMN epoch '',
COMMENT COLUMN epoch_start_date_time '',
COMMENT COLUMN meta_client_name '',
COMMENT COLUMN meta_client_id '',
COMMENT COLUMN meta_client_version '',
COMMENT COLUMN meta_client_implementation '',
COMMENT COLUMN meta_client_os '',
COMMENT COLUMN meta_client_ip '',
COMMENT COLUMN meta_client_geo_city '',
COMMENT COLUMN meta_client_geo_country '',
COMMENT COLUMN meta_client_geo_country_code '',
COMMENT COLUMN meta_client_geo_continent_code '',
COMMENT COLUMN meta_client_geo_longitude '',
COMMENT COLUMN meta_client_geo_latitude '',
COMMENT COLUMN meta_client_geo_autonomous_system_number '',
COMMENT COLUMN meta_client_geo_autonomous_system_organization '',
COMMENT COLUMN meta_network_id '',
COMMENT COLUMN meta_network_name '',
COMMENT COLUMN meta_consensus_version '',
COMMENT COLUMN meta_consensus_version_major '',
COMMENT COLUMN meta_consensus_version_minor '',
COMMENT COLUMN meta_consensus_version_patch '',
COMMENT COLUMN meta_consensus_implementation '',
COMMENT COLUMN meta_labels '';

ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT '';

ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT '';

ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}'
MODIFY COMMENT '';

ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}'
MODIFY COMMENT '';
