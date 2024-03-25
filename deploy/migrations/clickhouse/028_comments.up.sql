ALTER TABLE default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/beacon/states/{state_id}/committees data from each sentry client attached to a beacon node.',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API committee payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN committee_index 'The committee index in the beacon API committee payload',
COMMENT COLUMN validators 'The validator indices in the beacon API committee payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API committee payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN meta_client_id 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.',
COMMENT COLUMN meta_client_version 'Version of the client that generated the event',
COMMENT COLUMN meta_client_implementation 'Implementation of the client that generated the event',
COMMENT COLUMN meta_client_os 'Operating system of the client that generated the event',
COMMENT COLUMN meta_client_ip 'IP address of the client that generated the event',
COMMENT COLUMN meta_client_geo_city 'City of the client that generated the event',
COMMENT COLUMN meta_client_geo_country 'Country of the client that generated the event',
COMMENT COLUMN meta_client_geo_country_code 'Country code of the client that generated the event',
COMMENT COLUMN meta_client_geo_continent_code 'Continent code of the client that generated the event',
COMMENT COLUMN meta_client_geo_longitude 'Longitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_latitude 'Latitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_number 'Autonomous system number of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_organization 'Autonomous system organization of the client that generated the event',
COMMENT COLUMN meta_network_id 'Ethereum network ID',
COMMENT COLUMN meta_network_name 'Ethereum network name',
COMMENT COLUMN meta_consensus_version 'Ethereum consensus client version that generated the event',
COMMENT COLUMN meta_consensus_version_major 'Ethereum consensus client major version that generated the event',
COMMENT COLUMN meta_consensus_version_minor 'Ethereum consensus client minor version that generated the event',
COMMENT COLUMN meta_consensus_version_patch 'Ethereum consensus client patch version that generated the event',
COMMENT COLUMN meta_consensus_implementation 'Ethereum consensus client implementation that generated the event',
COMMENT COLUMN meta_labels 'Labels associated with the event';

ALTER TABLE default.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/beacon/states/{state_id}/committees data from each sentry client attached to a beacon node.',
COMMENT COLUMN event_date_time 'When the sentry received the event from a beacon node',
COMMENT COLUMN slot 'Slot number in the beacon API committee payload',
COMMENT COLUMN slot_start_date_time 'The wall clock time when the slot started',
COMMENT COLUMN committee_index 'The committee index in the beacon API committee payload',
COMMENT COLUMN validators 'The validator indices in the beacon API committee payload',
COMMENT COLUMN epoch 'The epoch number in the beacon API committee payload',
COMMENT COLUMN epoch_start_date_time 'The wall clock time when the epoch started',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN meta_client_id 'Unique Session ID of the client that generated the event. This changes every time the client is restarted.',
COMMENT COLUMN meta_client_version 'Version of the client that generated the event',
COMMENT COLUMN meta_client_implementation 'Implementation of the client that generated the event',
COMMENT COLUMN meta_client_os 'Operating system of the client that generated the event',
COMMENT COLUMN meta_client_ip 'IP address of the client that generated the event',
COMMENT COLUMN meta_client_geo_city 'City of the client that generated the event',
COMMENT COLUMN meta_client_geo_country 'Country of the client that generated the event',
COMMENT COLUMN meta_client_geo_country_code 'Country code of the client that generated the event',
COMMENT COLUMN meta_client_geo_continent_code 'Continent code of the client that generated the event',
COMMENT COLUMN meta_client_geo_longitude 'Longitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_latitude 'Latitude of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_number 'Autonomous system number of the client that generated the event',
COMMENT COLUMN meta_client_geo_autonomous_system_organization 'Autonomous system organization of the client that generated the event',
COMMENT COLUMN meta_network_id 'Ethereum network ID',
COMMENT COLUMN meta_network_name 'Ethereum network name',
COMMENT COLUMN meta_consensus_version 'Ethereum consensus client version that generated the event',
COMMENT COLUMN meta_consensus_version_major 'Ethereum consensus client major version that generated the event',
COMMENT COLUMN meta_consensus_version_minor 'Ethereum consensus client minor version that generated the event',
COMMENT COLUMN meta_consensus_version_patch 'Ethereum consensus client patch version that generated the event',
COMMENT COLUMN meta_consensus_implementation 'Ethereum consensus client implementation that generated the event',
COMMENT COLUMN meta_labels 'Labels associated with the event';

ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "blob_sidecar" data from each sentry client attached to a beacon node.';

ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "blob_sidecar" data from each sentry client attached to a beacon node.';

ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/validator/attestation_data data from each sentry client attached to a beacon node.';

ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/validator/attestation_data data from each sentry client attached to a beacon node.';
