-- Comprehensive table comment improvements across all Xatu ClickHouse tables
-- Improves table and column comments for succinctness, correctness, and partition key awareness
-- Covers: beacon_api_*, canonical_beacon_*, libp2p_*, mempool_*, mev_relay_*, node_record_*, and reference tables

-- ============================================================================
-- Batch 1: beacon_api_eth_v1_events_* table comments
-- ============================================================================

-- beacon_api_eth_v1_events_head
ALTER TABLE default.beacon_api_eth_v1_events_head ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures head events. Each row represents a `head` event from the Beacon API `/eth/v1/events?topics=head`, indicating the chain''s canonical head has been updated. Sentry adds client metadata and propagation timing. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_head_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures head events. Each row represents a `head` event from the Beacon API `/eth/v1/events?topics=head`, indicating the chain''s canonical head has been updated. Sentry adds client metadata and propagation timing. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

-- beacon_api_eth_v1_events_block
ALTER TABLE default.beacon_api_eth_v1_events_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures block events. Each row represents a `block` event from the Beacon API `/eth/v1/events?topics=block`, indicating a new block has been imported. Sentry adds client metadata and propagation timing. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures block events. Each row represents a `block` event from the Beacon API `/eth/v1/events?topics=block`, indicating a new block has been imported. Sentry adds client metadata and propagation timing. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

-- beacon_api_eth_v1_events_attestation
ALTER TABLE default.beacon_api_eth_v1_events_attestation ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures attestation events. Each row represents an `attestation` event from the Beacon API `/eth/v1/events?topics=attestation`. High-volume table - filter by `slot_start_date_time` and `meta_network_name` to avoid full scans. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_attestation_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures attestation events. Each row represents an `attestation` event from the Beacon API `/eth/v1/events?topics=attestation`. High-volume table - filter by `slot_start_date_time` and `meta_network_name` to avoid full scans. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

-- beacon_api_eth_v1_events_voluntary_exit
ALTER TABLE default.beacon_api_eth_v1_events_voluntary_exit ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures voluntary exit events. Each row represents a `voluntary_exit` event from the Beacon API `/eth/v1/events?topics=voluntary_exit`, when a validator initiates an exit. Partition: monthly by `wallclock_epoch_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients';

ALTER TABLE default.beacon_api_eth_v1_events_voluntary_exit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures voluntary exit events. Each row represents a `voluntary_exit` event from the Beacon API `/eth/v1/events?topics=voluntary_exit`, when a validator initiates an exit. Partition: monthly by `wallclock_epoch_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients';

-- beacon_api_eth_v1_events_finalized_checkpoint
ALTER TABLE default.beacon_api_eth_v1_events_finalized_checkpoint ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures finalized checkpoint events. Each row represents a `finalized_checkpoint` event from the Beacon API `/eth/v1/events?topics=finalized_checkpoint`, when the chain finalizes a new epoch. Partition: monthly by `epoch_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients';

ALTER TABLE default.beacon_api_eth_v1_events_finalized_checkpoint_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures finalized checkpoint events. Each row represents a `finalized_checkpoint` event from the Beacon API `/eth/v1/events?topics=finalized_checkpoint`, when the chain finalizes a new epoch. Partition: monthly by `epoch_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients';

-- beacon_api_eth_v1_events_chain_reorg
ALTER TABLE default.beacon_api_eth_v1_events_chain_reorg ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures chain reorg events. Each row represents a `chain_reorg` event from the Beacon API `/eth/v1/events?topics=chain_reorg`, when the beacon node detects a chain reorganization. Includes depth and old/new head info. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_chain_reorg_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures chain reorg events. Each row represents a `chain_reorg` event from the Beacon API `/eth/v1/events?topics=chain_reorg`, when the beacon node detects a chain reorganization. Includes depth and old/new head info. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

-- beacon_api_eth_v1_events_contribution_and_proof
ALTER TABLE default.beacon_api_eth_v1_events_contribution_and_proof ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures sync committee contribution events. Each row represents a `contribution_and_proof` event from the Beacon API `/eth/v1/events?topics=contribution_and_proof`. Partition: monthly by `contribution_slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN contribution_propagation_slot_start_diff 'Time in milliseconds since the start of the contribution slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_contribution_and_proof_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures sync committee contribution events. Each row represents a `contribution_and_proof` event from the Beacon API `/eth/v1/events?topics=contribution_and_proof`. Partition: monthly by `contribution_slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN contribution_propagation_slot_start_diff 'Time in milliseconds since the start of the contribution slot when the Sentry received this event';

-- beacon_api_eth_v1_events_blob_sidecar
ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures blob sidecar events. Each row represents a `blob_sidecar` event from the Beacon API `/eth/v1/events?topics=blob_sidecar` (EIP-4844) with KZG commitment data. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures blob sidecar events. Each row represents a `blob_sidecar` event from the Beacon API `/eth/v1/events?topics=blob_sidecar` (EIP-4844) with KZG commitment data. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

-- beacon_api_eth_v1_events_data_column_sidecar
ALTER TABLE default.beacon_api_eth_v1_events_data_column_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures data column sidecar events. Each row represents a `data_column_sidecar` event from the Beacon API `/eth/v1/events?topics=data_column_sidecar` (PeerDAS) with data availability sampling info. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_data_column_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures data column sidecar events. Each row represents a `data_column_sidecar` event from the Beacon API `/eth/v1/events?topics=data_column_sidecar` (PeerDAS) with data availability sampling info. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

-- beacon_api_eth_v1_events_block_gossip
ALTER TABLE default.beacon_api_eth_v1_events_block_gossip ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures block gossip events. Each row represents a `block_gossip` event from the Beacon API `/eth/v1/events?topics=block_gossip` used for measuring block propagation timing across the network. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';

ALTER TABLE default.beacon_api_eth_v1_events_block_gossip_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry subscribes to a beacon node''s Beacon API event-stream and captures block gossip events. Each row represents a `block_gossip` event from the Beacon API `/eth/v1/events?topics=block_gossip` used for measuring block propagation timing across the network. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the event. The table contains data from multiple Sentry clients',
COMMENT COLUMN propagation_slot_start_diff 'Time in milliseconds since the start of the slot when the Sentry received this event';
-- Batch 2: beacon_api_eth_v1_* and beacon_api_eth_v2_* table comments (non-event-stream tables)
-- Improves table and column comments for succinctness, correctness, and partition key awareness

-- beacon_api_eth_v1_beacon_committee
ALTER TABLE default.beacon_api_eth_v1_beacon_committee ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v1/beacon/states/{state_id}/committees` endpoint to fetch committee assignments. Each row contains validator committee assignments for a slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

ALTER TABLE default.beacon_api_eth_v1_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v1/beacon/states/{state_id}/committees` endpoint to fetch committee assignments. Each row contains validator committee assignments for a slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

-- beacon_api_eth_v1_validator_attestation_data
ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v1/validator/attestation_data` endpoint to fetch attestation data. Each row contains attestation data with request timing metrics. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients',
COMMENT COLUMN request_duration 'Time in milliseconds for the Beacon API request to complete',
COMMENT COLUMN request_slot_start_diff 'Time in milliseconds since the start of the slot when the request was made';

ALTER TABLE default.beacon_api_eth_v1_validator_attestation_data_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v1/validator/attestation_data` endpoint to fetch attestation data. Each row contains attestation data with request timing metrics. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients',
COMMENT COLUMN request_duration 'Time in milliseconds for the Beacon API request to complete',
COMMENT COLUMN request_slot_start_diff 'Time in milliseconds since the start of the slot when the request was made';

-- beacon_api_eth_v1_proposer_duty
ALTER TABLE default.beacon_api_eth_v1_proposer_duty ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry fetches proposer duties from the Beacon API `/eth/v1/validator/duties/proposer/{epoch}` endpoint. Each row contains which validator is scheduled to propose a block for a given slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

ALTER TABLE default.beacon_api_eth_v1_proposer_duty_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry fetches proposer duties from the Beacon API `/eth/v1/validator/duties/proposer/{epoch}` endpoint. Each row contains which validator is scheduled to propose a block for a given slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

-- beacon_api_eth_v2_beacon_block
ALTER TABLE default.beacon_api_eth_v2_beacon_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v2/beacon/blocks/{block_id}` endpoint to fetch beacon blocks. Each row contains full beacon block data including execution payload. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

ALTER TABLE default.beacon_api_eth_v2_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v2/beacon/blocks/{block_id}` endpoint to fetch beacon blocks. Each row contains full beacon block data including execution payload. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

-- beacon_api_eth_v3_validator_block
ALTER TABLE default.beacon_api_eth_v3_validator_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v3/validator/blocks/{slot}` endpoint. Contains UNSIGNED simulated blocks - what the beacon node would have built if asked to propose at that slot. NOT actual proposed blocks. Useful for MEV research and block building analysis. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';

ALTER TABLE default.beacon_api_eth_v3_validator_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Xatu Sentry calls the Beacon API `/eth/v3/validator/blocks/{slot}` endpoint. Contains UNSIGNED simulated blocks - what the beacon node would have built if asked to propose at that slot. NOT actual proposed blocks. Useful for MEV research and block building analysis. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the Sentry client that collected the data. The table contains data from multiple Sentry clients';
-- Batch 3: canonical_beacon_block* table comments
-- Improves table and column comments for succinctness, correctness, and partition key awareness

-- canonical_beacon_block
ALTER TABLE default.canonical_beacon_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized beacon block data. Each row represents a canonical block. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized beacon block data. Each row represents a canonical block. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_block_attester_slashing
ALTER TABLE default.canonical_beacon_block_attester_slashing ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains attester slashings from finalized beacon blocks. Each row represents two conflicting attestations from a slashed validator. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_block_attester_slashing_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains attester slashings from finalized beacon blocks. Each row represents two conflicting attestations from a slashed validator. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_block_deposit
ALTER TABLE default.canonical_beacon_block_deposit ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains validator deposits from finalized beacon blocks. Each row represents a deposit with pubkey, withdrawal credentials, and amount. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_block_deposit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains validator deposits from finalized beacon blocks. Each row represents a deposit with pubkey, withdrawal credentials, and amount. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_block_execution_transaction
ALTER TABLE default.canonical_beacon_block_execution_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution layer transactions from finalized beacon blocks. Each row represents a transaction from the execution payload. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_block_execution_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution layer transactions from finalized beacon blocks. Each row represents a transaction from the execution payload. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_block_voluntary_exit
ALTER TABLE default.canonical_beacon_block_voluntary_exit ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains voluntary exits from finalized beacon blocks. Each row represents a validator initiating an exit. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_block_voluntary_exit_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains voluntary exits from finalized beacon blocks. Each row represents a validator initiating an exit. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_block_withdrawal
ALTER TABLE default.canonical_beacon_block_withdrawal ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains withdrawals from finalized beacon blocks. Each row represents a validator withdrawal with recipient address and amount. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_block_withdrawal_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains withdrawals from finalized beacon blocks. Each row represents a validator withdrawal with recipient address and amount. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';
-- Batch 4: canonical_beacon_* remaining table comments
-- Improves table and column comments for succinctness, correctness, and partition key awareness

-- canonical_beacon_validators
ALTER TABLE default.canonical_beacon_validators ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized validator state snapshots. Each row represents a validator''s status, balance, and lifecycle epochs at a specific epoch. Partition: monthly by `epoch_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_validators_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized validator state snapshots. Each row represents a validator''s status, balance, and lifecycle epochs at a specific epoch. Partition: monthly by `epoch_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_committee
ALTER TABLE default.canonical_beacon_committee ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized beacon committee assignments. Each row represents a committee with its validator indices for a given slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_committee_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized beacon committee assignments. Each row represents a committee with its validator indices for a given slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_proposer_duty
ALTER TABLE default.canonical_beacon_proposer_duty ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized proposer duty assignments. Each row represents which validator was scheduled to propose a block for a given slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_proposer_duty_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains finalized proposer duty assignments. Each row represents which validator was scheduled to propose a block for a given slot. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_elaborated_attestation
ALTER TABLE default.canonical_beacon_elaborated_attestation ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains elaborated attestations from finalized beacon blocks. Aggregation bits are expanded to actual validator indices. Each row represents an attestation with its participating validators, source/target checkpoints, and position in the block. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_elaborated_attestation_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains elaborated attestations from finalized beacon blocks. Aggregation bits are expanded to actual validator indices. Each row represents an attestation with its participating validators, source/target checkpoints, and position in the block. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- canonical_beacon_blob_sidecar
ALTER TABLE default.canonical_beacon_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains blob sidecars from finalized beacon blocks. Each row represents a blob with its KZG commitment, proof, and versioned hash. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.canonical_beacon_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains blob sidecars from finalized beacon blocks. Each row represents a blob with its KZG commitment, proof, and versioned hash. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- Batch 5a: libp2p_gossipsub_* table comments
-- Tables populated from deep instrumentation within consensus layer clients (forked Prysm/Lighthouse maintained by ethPandaOps)

-- libp2p_gossipsub_beacon_block
ALTER TABLE default.libp2p_gossipsub_beacon_block ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon block messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients (Prysm/Lighthouse). Each row represents a block gossiped on the p2p network with timing and peer metadata. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_gossipsub_beacon_block_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon block messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients (Prysm/Lighthouse). Each row represents a block gossiped on the p2p network with timing and peer metadata. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_gossipsub_beacon_attestation
ALTER TABLE default.libp2p_gossipsub_beacon_attestation ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon attestation messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents an attestation gossiped on the p2p network with timing and peer metadata. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_gossipsub_beacon_attestation_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains beacon attestation messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents an attestation gossiped on the p2p network with timing and peer metadata. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_gossipsub_blob_sidecar
ALTER TABLE default.libp2p_gossipsub_blob_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains blob sidecar messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents a blob gossiped on the p2p network with timing and peer metadata. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_gossipsub_blob_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains blob sidecar messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents a blob gossiped on the p2p network with timing and peer metadata. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_gossipsub_data_column_sidecar
ALTER TABLE default.libp2p_gossipsub_data_column_sidecar ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains data column sidecar messages received via libp2p gossipsub (PeerDAS). Collected from deep instrumentation within forked consensus layer clients. Each row represents a data column gossiped on the p2p network. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_gossipsub_data_column_sidecar_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains data column sidecar messages received via libp2p gossipsub (PeerDAS). Collected from deep instrumentation within forked consensus layer clients. Each row represents a data column gossiped on the p2p network. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_gossipsub_aggregate_and_proof
ALTER TABLE default.libp2p_gossipsub_aggregate_and_proof ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains aggregate and proof messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents an aggregated attestation with its proof. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_gossipsub_aggregate_and_proof_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains aggregate and proof messages received via libp2p gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents an aggregated attestation with its proof. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- Batch 5b: libp2p peer/connection table comments
-- Tables populated from deep instrumentation within consensus layer clients (forked Prysm/Lighthouse maintained by ethPandaOps)

-- libp2p_peer
ALTER TABLE default.libp2p_peer ON CLUSTER '{cluster}'
MODIFY COMMENT 'Lookup table mapping seahashed peer_id + network to original peer ID. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.';

ALTER TABLE default.libp2p_peer_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Lookup table mapping seahashed peer_id + network to original peer ID. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.';

-- libp2p_add_peer
ALTER TABLE default.libp2p_add_peer ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains ADD_PEER events when peers are added to the libp2p peer store. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_add_peer_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains ADD_PEER events when peers are added to the libp2p peer store. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_remove_peer
ALTER TABLE default.libp2p_remove_peer ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains REMOVE_PEER events when peers are removed from the libp2p peer store. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_remove_peer_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains REMOVE_PEER events when peers are removed from the libp2p peer store. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_connected
ALTER TABLE default.libp2p_connected ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains CONNECTED events when connections are established to remote peers. Collected from deep instrumentation within forked consensus layer clients. Each row includes remote peer agent info and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_connected_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains CONNECTED events when connections are established to remote peers. Collected from deep instrumentation within forked consensus layer clients. Each row includes remote peer agent info and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_disconnected
ALTER TABLE default.libp2p_disconnected ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains DISCONNECTED events when connections to remote peers are closed. Collected from deep instrumentation within forked consensus layer clients. Each row includes remote peer agent info and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_disconnected_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains DISCONNECTED events when connections to remote peers are closed. Collected from deep instrumentation within forked consensus layer clients. Each row includes remote peer agent info and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_handle_status
ALTER TABLE default.libp2p_handle_status ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains status protocol handling events (req/resp). Collected from deep instrumentation within forked consensus layer clients. Each row represents a status exchange with a peer including their head and finalized info. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_handle_status_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains status protocol handling events (req/resp). Collected from deep instrumentation within forked consensus layer clients. Each row represents a status exchange with a peer including their head and finalized info. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_handle_metadata
ALTER TABLE default.libp2p_handle_metadata ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains metadata protocol handling events (req/resp). Collected from deep instrumentation within forked consensus layer clients. Each row represents a metadata exchange with a peer including their attnets and syncnets. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_handle_metadata_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains metadata protocol handling events (req/resp). Collected from deep instrumentation within forked consensus layer clients. Each row represents a metadata exchange with a peer including their attnets and syncnets. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_synthetic_heartbeat
ALTER TABLE default.libp2p_synthetic_heartbeat ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains periodic heartbeat snapshots of libp2p peer state. Collected from deep instrumentation within forked consensus layer clients. Each row contains mesh/peer counts and topic subscriptions at that moment. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_synthetic_heartbeat_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains periodic heartbeat snapshots of libp2p peer state. Collected from deep instrumentation within forked consensus layer clients. Each row contains mesh/peer counts and topic subscriptions at that moment. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- Batch 5c: libp2p RPC table comments
-- Tables populated from deep instrumentation within consensus layer clients (forked Prysm/Lighthouse maintained by ethPandaOps)

-- libp2p_recv_rpc
ALTER TABLE default.libp2p_recv_rpc ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC messages received from peers. Collected from deep instrumentation within forked consensus layer clients. Control messages are split into separate tables referencing this via rpc_meta_unique_key. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_recv_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC messages received from peers. Collected from deep instrumentation within forked consensus layer clients. Control messages are split into separate tables referencing this via rpc_meta_unique_key. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_send_rpc
ALTER TABLE default.libp2p_send_rpc ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC messages sent to peers. Collected from deep instrumentation within forked consensus layer clients. Control messages are split into separate tables referencing this via rpc_meta_unique_key. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_send_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC messages sent to peers. Collected from deep instrumentation within forked consensus layer clients. Control messages are split into separate tables referencing this via rpc_meta_unique_key. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_drop_rpc
ALTER TABLE default.libp2p_drop_rpc ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC messages dropped (not processed) by the peer. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_drop_rpc_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC messages dropped (not processed) by the peer. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_message
ALTER TABLE default.libp2p_rpc_meta_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC message metadata from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents a message within an RPC with topic and message ID. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC message metadata from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents a message within an RPC with topic and message ID. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_subscription
ALTER TABLE default.libp2p_rpc_meta_subscription ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC subscription changes from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents a subscribe/unsubscribe action for a topic. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_subscription_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains RPC subscription changes from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Each row represents a subscribe/unsubscribe action for a topic. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_control_ihave
ALTER TABLE default.libp2p_rpc_meta_control_ihave ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains IHAVE control messages from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Peers advertise message IDs they have available. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_control_ihave_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains IHAVE control messages from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Peers advertise message IDs they have available. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_control_iwant
ALTER TABLE default.libp2p_rpc_meta_control_iwant ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains IWANT control messages from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Peers request specific message IDs. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_control_iwant_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains IWANT control messages from gossipsub. Collected from deep instrumentation within forked consensus layer clients. Peers request specific message IDs. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_control_graft
ALTER TABLE default.libp2p_rpc_meta_control_graft ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains GRAFT control messages from gossipsub RPC. Collected from deep instrumentation within forked consensus layer clients. Peers request to join the mesh for a topic. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_control_graft_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains GRAFT control messages from gossipsub RPC. Collected from deep instrumentation within forked consensus layer clients. Peers request to join the mesh for a topic. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_control_prune
ALTER TABLE default.libp2p_rpc_meta_control_prune ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains PRUNE control messages from gossipsub RPC. Collected from deep instrumentation within forked consensus layer clients. Peers are removed from the mesh for a topic. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_control_prune_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains PRUNE control messages from gossipsub RPC. Collected from deep instrumentation within forked consensus layer clients. Peers are removed from the mesh for a topic. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_rpc_meta_control_idontwant
ALTER TABLE default.libp2p_rpc_meta_control_idontwant ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains IDONTWANT control messages from gossipsub RPC. Collected from deep instrumentation within forked consensus layer clients. Peers indicate they do not want certain messages. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_rpc_meta_control_idontwant_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains IDONTWANT control messages from gossipsub RPC. Collected from deep instrumentation within forked consensus layer clients. Peers indicate they do not want certain messages. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- Batch 5d: libp2p topic/message event table comments
-- Tables populated from deep instrumentation within consensus layer clients (forked Prysm/Lighthouse maintained by ethPandaOps)

-- libp2p_join
ALTER TABLE default.libp2p_join ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains JOIN events when the local node joins a gossipsub topic. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_join_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains JOIN events when the local node joins a gossipsub topic. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_leave
ALTER TABLE default.libp2p_leave ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains LEAVE events when the local node leaves a gossipsub topic. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_leave_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains LEAVE events when the local node leaves a gossipsub topic. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_graft
ALTER TABLE default.libp2p_graft ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains GRAFT events when a peer joins the mesh for a topic. Collected from deep instrumentation within forked consensus layer clients. Tracks mesh membership changes. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_graft_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains GRAFT events when a peer joins the mesh for a topic. Collected from deep instrumentation within forked consensus layer clients. Tracks mesh membership changes. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_prune
ALTER TABLE default.libp2p_prune ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains PRUNE events when a peer is removed from the mesh for a topic. Collected from deep instrumentation within forked consensus layer clients. Tracks mesh membership changes. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_prune_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains PRUNE events when a peer is removed from the mesh for a topic. Collected from deep instrumentation within forked consensus layer clients. Tracks mesh membership changes. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_deliver_message
ALTER TABLE default.libp2p_deliver_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains DELIVER_MESSAGE events when messages are delivered to local subscribers. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_deliver_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains DELIVER_MESSAGE events when messages are delivered to local subscribers. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_publish_message
ALTER TABLE default.libp2p_publish_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains PUBLISH_MESSAGE events when the local node publishes messages to gossipsub. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_publish_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains PUBLISH_MESSAGE events when the local node publishes messages to gossipsub. Collected from deep instrumentation within forked consensus layer clients. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_duplicate_message
ALTER TABLE default.libp2p_duplicate_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains DUPLICATE_MESSAGE events when a message is received that was already seen. Collected from deep instrumentation within forked consensus layer clients. Useful for analyzing message propagation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_duplicate_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains DUPLICATE_MESSAGE events when a message is received that was already seen. Collected from deep instrumentation within forked consensus layer clients. Useful for analyzing message propagation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- libp2p_reject_message
ALTER TABLE default.libp2p_reject_message ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains REJECT_MESSAGE events when messages fail validation and are rejected. Collected from deep instrumentation within forked consensus layer clients. Includes rejection reason. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.libp2p_reject_message_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains REJECT_MESSAGE events when messages fail validation and are rejected. Collected from deep instrumentation within forked consensus layer clients. Includes rejection reason. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- Batch 6: mempool_*, mev_relay_*, node_record_* table comments

-- mempool_transaction
ALTER TABLE default.mempool_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains pending transactions observed in the mempool. Each row represents a transaction first seen at a specific time with its gas parameters. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.mempool_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains pending transactions observed in the mempool. Each row represents a transaction first seen at a specific time with its gas parameters. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- mempool_dumpster_transaction
ALTER TABLE default.mempool_dumpster_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions imported from the external Mempool Dumpster dataset. Historical mempool data following the parquet schema with additions. Partition: monthly by `event_date_time`.';

ALTER TABLE default.mempool_dumpster_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions imported from the external Mempool Dumpster dataset. Historical mempool data following the parquet schema with additions. Partition: monthly by `event_date_time`.';

-- block_native_mempool_transaction
ALTER TABLE default.block_native_mempool_transaction ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions imported from the external Blocknative mempool dataset. Partition: monthly by `event_date_time`.';

ALTER TABLE default.block_native_mempool_transaction_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains transactions imported from the external Blocknative mempool dataset. Partition: monthly by `event_date_time`.';

-- mev_relay_bid_trace
ALTER TABLE default.mev_relay_bid_trace ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains block bids collected by polling MEV relay data APIs. Each row represents a bid from a builder to a relay with value and block details. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.mev_relay_bid_trace_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains block bids collected by polling MEV relay data APIs. Each row represents a bid from a builder to a relay with value and block details. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- mev_relay_proposer_payload_delivered
ALTER TABLE default.mev_relay_proposer_payload_delivered ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains delivered payloads collected by polling MEV relay data APIs. Each row represents a payload that was delivered to a proposer. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.mev_relay_proposer_payload_delivered_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains delivered payloads collected by polling MEV relay data APIs. Each row represents a payload that was delivered to a proposer. Partition: monthly by `slot_start_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- mev_relay_validator_registration
ALTER TABLE default.mev_relay_validator_registration ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains validator registrations collected by polling MEV relay data APIs. Each row represents a validator registering their fee recipient and gas limit preferences. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.mev_relay_validator_registration_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains validator registrations collected by polling MEV relay data APIs. Each row represents a validator registering their fee recipient and gas limit preferences. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- node_record_execution
ALTER TABLE default.node_record_execution ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution layer node records discovered via discv5 network crawling. Each row represents a discovered node with its ENR, client info, and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.node_record_execution_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains execution layer node records discovered via discv5 network crawling. Each row represents a discovered node with its ENR, client info, and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- node_record_consensus
ALTER TABLE default.node_record_consensus ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains consensus layer node records discovered via discv5 network crawling. Each row represents a discovered node with its ENR, client info, and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

ALTER TABLE default.node_record_consensus_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Contains consensus layer node records discovered via discv5 network crawling. Each row represents a discovered node with its ENR, client info, and geolocation. Partition: monthly by `event_date_time`.',
COMMENT COLUMN meta_client_name 'Name of the client that collected the data. The table contains data from multiple clients';

-- Batch 7: Remaining table comments

-- blob_submitter
ALTER TABLE default.blob_submitter ON CLUSTER '{cluster}'
MODIFY COMMENT 'Lookup table mapping blob submitter addresses to names. Typically used to identify L2 sequencers and rollups submitting blobs to Ethereum. Partition: none (reference table).';

ALTER TABLE default.blob_submitter_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Lookup table mapping blob submitter addresses to names. Typically used to identify L2 sequencers and rollups submitting blobs to Ethereum. Partition: none (reference table).';

-- ethseer_validator_entity
ALTER TABLE default.ethseer_validator_entity ON CLUSTER '{cluster}'
MODIFY COMMENT 'Lookup table mapping validators to entities, imported from Ethseer. Used to label validators by their staking provider or operator. Partition: none (reference table).';

ALTER TABLE default.ethseer_validator_entity_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Lookup table mapping validators to entities, imported from Ethseer. Used to label validators by their staking provider or operator. Partition: none (reference table).';

