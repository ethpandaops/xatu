-- Rollback: Restore original comments for all tables
-- This reverses the comprehensive table comment improvements

-- ============================================================================
-- Rollback: beacon_api_eth_v1_events_* tables
-- ============================================================================

-- beacon_api_eth_v1_events_head
ALTER TABLE default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "head" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

ALTER TABLE default.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "head" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

-- beacon_api_eth_v1_events_block
ALTER TABLE default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "block" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

ALTER TABLE default.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "block" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

-- beacon_api_eth_v1_events_attestation
ALTER TABLE default.beacon_api_eth_v1_events_attestation ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "attestation" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

ALTER TABLE default.beacon_api_eth_v1_events_attestation_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "attestation" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

-- beacon_api_eth_v1_events_voluntary_exit
ALTER TABLE default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "voluntary exit" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "voluntary exit" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- beacon_api_eth_v1_events_finalized_checkpoint
ALTER TABLE default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "finalized checkpoint" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "finalized checkpoint" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- beacon_api_eth_v1_events_chain_reorg
ALTER TABLE default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "chain reorg" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'Difference in slots between when the reorg occurred and when the sentry received the event';

ALTER TABLE default.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "chain reorg" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'Difference in slots between when the reorg occurred and when the sentry received the event';

-- beacon_api_eth_v1_events_contribution_and_proof
ALTER TABLE default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "contribution and proof" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN contribution_propagation_slot_start_diff 'Difference in slots between when the contribution occurred and when the sentry received the event';

ALTER TABLE default.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "contribution and proof" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN contribution_propagation_slot_start_diff 'Difference in slots between when the contribution occurred and when the sentry received the event';

-- beacon_api_eth_v1_events_blob_sidecar
ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "blob_sidecar" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "blob_sidecar" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

-- beacon_api_eth_v1_events_data_column_sidecar
ALTER TABLE default.beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "data_column_sidecar" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

ALTER TABLE default.beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "data_column_sidecar" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

-- beacon_api_eth_v1_events_block_gossip
ALTER TABLE default.beacon_api_eth_v1_events_block_gossip ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "block_gossip" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';

ALTER TABLE default.beacon_api_eth_v1_events_block_gossip_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API eventstream "block_gossip" data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN propagation_slot_start_diff 'The difference between the event_date_time and the slot_start_date_time';
-- Rollback: Restore original comments for beacon_api_eth_v1_* and beacon_api_eth_v2_* tables (non-event-stream)

-- beacon_api_eth_v1_beacon_committee
ALTER TABLE default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/beacon/states/{state_id}/committees data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/beacon/states/{state_id}/committees data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- beacon_api_eth_v1_validator_attestation_data
ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/validator/attestation_data data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN request_duration 'The duration of the request',
COMMENT COLUMN request_slot_start_diff 'The difference between the request time and the slot start time';

ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v1/validator/attestation_data data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event',
COMMENT COLUMN request_duration 'The duration of the request',
COMMENT COLUMN request_slot_start_diff 'The difference between the request time and the slot start time';

-- beacon_api_eth_v1_proposer_duty
ALTER TABLE default.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a proposer duty from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.beacon_api_eth_v1_proposer_duty_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a proposer duty from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- beacon_api_eth_v2_beacon_block
ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v2/beacon/blocks/{block_id} data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v2/beacon/blocks/{block_id} data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- beacon_api_eth_v3_validator_block
ALTER TABLE default.beacon_api_eth_v3_validator_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v3/validator/blocks/{slot} data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.beacon_api_eth_v3_validator_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon API /eth/v3/validator/blocks/{slot} data from each sentry client attached to a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';
-- Rollback: Restore original comments for canonical_beacon_block* tables

-- canonical_beacon_block
ALTER TABLE default.canonical_beacon_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon block from a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon block from a beacon node.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_block_attester_slashing
ALTER TABLE default.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains attester slashing from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_block_attester_slashing_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains attester slashing from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_block_deposit
ALTER TABLE default.canonical_beacon_block_deposit ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a deposit from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_block_deposit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a deposit from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_block_execution_transaction
ALTER TABLE default.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution transaction from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution transaction from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_block_voluntary_exit
ALTER TABLE default.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a voluntary exit from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_block_voluntary_exit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a voluntary exit from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_block_withdrawal
ALTER TABLE default.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a withdrawal from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a withdrawal from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';
-- Rollback: Restore original comments for canonical_beacon_* remaining tables

-- canonical_beacon_validators
ALTER TABLE default.canonical_beacon_validators ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a validator state for an epoch.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_validators_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a validator state for an epoch.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_committee
ALTER TABLE default.canonical_beacon_committee ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains canonical beacon API /eth/v1/beacon/states/{state_id}/committees data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains canonical beacon API /eth/v1/beacon/states/{state_id}/committees data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_proposer_duty
ALTER TABLE default.canonical_beacon_proposer_duty ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a proposer duty from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_proposer_duty_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a proposer duty from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_elaborated_attestation
ALTER TABLE default.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains elaborated attestations from beacon blocks.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains elaborated attestations from beacon blocks.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- canonical_beacon_blob_sidecar
ALTER TABLE default.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a blob sidecar from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.canonical_beacon_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a blob sidecar from a beacon block.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- Rollback: Restore original comments for libp2p_gossipsub_* tables

-- libp2p_gossipsub_beacon_block
ALTER TABLE default.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub beacon block data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub beacon block data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_gossipsub_beacon_attestation
ALTER TABLE default.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub beacon attestation data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub beacon attestation data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_gossipsub_blob_sidecar
ALTER TABLE default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub blob sidecar data',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub blob sidecar data',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_gossipsub_data_column_sidecar
ALTER TABLE default.libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub data column sidecar data',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub data column sidecar data',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_gossipsub_aggregate_and_proof
ALTER TABLE default.libp2p_gossipsub_aggregate_and_proof ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub aggregate and proof data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_gossipsub_aggregate_and_proof_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Table for libp2p gossipsub aggregate and proof data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- Rollback: Restore original comments for libp2p peer/connection tables

-- libp2p_peer
ALTER TABLE default.libp2p_peer ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the original peer id of a seahashed peer_id + meta_network_name, commonly seen in other tables as the field peer_id_unique_key',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_peer_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the original peer id of a seahashed peer_id + meta_network_name, commonly seen in other tables as the field peer_id_unique_key',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_add_peer
ALTER TABLE default.libp2p_add_peer ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers added to the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_add_peer_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers added to the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_remove_peer
ALTER TABLE default.libp2p_remove_peer ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers removed from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_remove_peer_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the peers removed from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_connected
ALTER TABLE default.libp2p_connected ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the CONNECTED events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_connected_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the CONNECTED events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_disconnected
ALTER TABLE default.libp2p_disconnected ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DISCONNECTED events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_disconnected_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DISCONNECTED events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_handle_status
ALTER TABLE default.libp2p_handle_status ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the status handling events for libp2p peers.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_handle_status_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the status handling events for libp2p peers.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_handle_metadata
ALTER TABLE default.libp2p_handle_metadata ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the metadata handling events for libp2p peers.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_handle_metadata_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the metadata handling events for libp2p peers.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_synthetic_heartbeat
ALTER TABLE default.libp2p_synthetic_heartbeat ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains heartbeat events from libp2p peers',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_synthetic_heartbeat_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains heartbeat events from libp2p peers',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- Rollback: Restore original comments for libp2p RPC tables

-- libp2p_recv_rpc
ALTER TABLE default.libp2p_recv_rpc ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages received by the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_recv_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages received by the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_send_rpc
ALTER TABLE default.libp2p_send_rpc ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages sent by the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_send_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages sent by the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_drop_rpc
ALTER TABLE default.libp2p_drop_rpc ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages dropped by the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_drop_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC messages dropped by the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_message
ALTER TABLE default.libp2p_rpc_meta_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC meta messages from the peer',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC meta messages from the peer',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_subscription
ALTER TABLE default.libp2p_rpc_meta_subscription ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC subscriptions from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the RPC subscriptions from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_control_ihave
ALTER TABLE default.libp2p_rpc_meta_control_ihave ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I have" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I have" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_control_iwant
ALTER TABLE default.libp2p_rpc_meta_control_iwant ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I want" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "I want" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_control_graft
ALTER TABLE default.libp2p_rpc_meta_control_graft ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Graft" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Graft" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_control_prune
ALTER TABLE default.libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Prune" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the "Prune" control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_rpc_meta_control_idontwant
ALTER TABLE default.libp2p_rpc_meta_control_idontwant ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the IDONTWANT control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_rpc_meta_control_idontwant_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the IDONTWANT control messages from the peer.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- Rollback: Restore original comments for libp2p topic/message event tables

-- libp2p_join
ALTER TABLE default.libp2p_join ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the JOIN events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_join_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the JOIN events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_leave
ALTER TABLE default.libp2p_leave ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the LEAVE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_leave_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the LEAVE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_graft
ALTER TABLE default.libp2p_graft ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the GRAFT events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_graft_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the GRAFT events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_prune
ALTER TABLE default.libp2p_prune ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the PRUNE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_prune_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the PRUNE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_deliver_message
ALTER TABLE default.libp2p_deliver_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DELIVER_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_deliver_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DELIVER_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_publish_message
ALTER TABLE default.libp2p_publish_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the PUBLISH_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_publish_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the PUBLISH_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_duplicate_message
ALTER TABLE default.libp2p_duplicate_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DUPLICATE_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_duplicate_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the DUPLICATE_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- libp2p_reject_message
ALTER TABLE default.libp2p_reject_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the REJECT_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.libp2p_reject_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains the details of the REJECT_MESSAGE events from the libp2p client.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- Rollback: Restore original comments for mempool_*, mev_relay_*, node_record_* tables

-- mempool_transaction (no original COMMENT)
-- Leaving unchanged as there was no original table comment

-- mempool_dumpster_transaction
ALTER TABLE default.mempool_dumpster_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from mempool dumpster dataset. Following the parquet schema with some additions',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.mempool_dumpster_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from mempool dumpster dataset. Following the parquet schema with some additions',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- block_native_mempool_transaction
ALTER TABLE default.block_native_mempool_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from block native mempool dataset',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.block_native_mempool_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions from block native mempool dataset',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- mev_relay_bid_trace
ALTER TABLE default.mev_relay_bid_trace ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains MEV relay block bids data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.mev_relay_bid_trace_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains MEV relay block bids data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- mev_relay_proposer_payload_delivered
ALTER TABLE default.mev_relay_proposer_payload_delivered ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains MEV relay proposer payload delivered data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.mev_relay_proposer_payload_delivered_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains MEV relay proposer payload delivered data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- mev_relay_validator_registration
ALTER TABLE default.mev_relay_validator_registration ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains MEV relay validator registrations data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.mev_relay_validator_registration_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains MEV relay validator registrations data.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- node_record_execution
ALTER TABLE default.node_record_execution ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution node records discovered by the Xatu discovery module.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.node_record_execution_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution node records discovered by the Xatu discovery module.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- node_record_consensus
ALTER TABLE default.node_record_consensus ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains consensus node records discovered by the Xatu discovery module.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

ALTER TABLE default.node_record_consensus_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains consensus node records discovered by the Xatu discovery module.',
COMMENT COLUMN meta_client_name 'Name of the client that generated the event';

-- Rollback: Restore original comments for remaining tables

-- blob_submitter
ALTER TABLE default.blob_submitter ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains blob submitter address to name mappings.';

ALTER TABLE default.blob_submitter_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains blob submitter address to name mappings.';

-- ethseer_validator_entity
ALTER TABLE default.ethseer_validator_entity ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a mapping of validators to entities';

ALTER TABLE default.ethseer_validator_entity_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains a mapping of validators to entities';

