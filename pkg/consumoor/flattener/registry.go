package flattener

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// All returns all registered flatteners used by consumoor.
func All() []Flattener {
	flatteners := []Flattener{
		// Phase 1
		NewGenericFlattener("beacon_api_eth_v1_events_head", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_block", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_block_gossip", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_finalized_checkpoint", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_chain_reorg", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_beacon_blob", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
		}, nil, nil, nil),
		NewGenericFlattener("execution_state_size", []xatu.Event_Name{
			xatu.Event_EXECUTION_STATE_SIZE,
		}, nil, nil, nil),

		// Phase 2
		NewGenericFlattener("beacon_api_eth_v1_events_blob_sidecar", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_data_column_sidecar", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_voluntary_exit", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_contribution_and_proof", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_events_attestation", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_validator_attestation_data", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
		}, nil, nil, nil),
		NewGenericFlattener("mempool_transaction", []xatu.Event_Name{
			xatu.Event_MEMPOOL_TRANSACTION,
			xatu.Event_MEMPOOL_TRANSACTION_V2,
		}, nil, nil, nil),
		NewGenericFlattener("execution_engine_new_payload", []xatu.Event_Name{
			xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
		}, nil, nil, nil),
		NewGenericFlattener("execution_engine_get_blobs", []xatu.Event_Name{
			xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
		}, nil, nil, nil),
		NewGenericFlattener("execution_block_metrics", []xatu.Event_Name{
			xatu.Event_EXECUTION_BLOCK_METRICS,
		}, nil, nil, nil),
		NewGenericFlattener("consensus_engine_api_new_payload", []xatu.Event_Name{
			xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
		}, nil, nil, nil),
		NewGenericFlattener("consensus_engine_api_get_blobs", []xatu.Event_Name{
			xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_blob_sidecar", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
		}, nil, nil, nil),
		NewGenericFlattener("beacon_api_eth_v1_proposer_duty", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
		}, stringFromAdditionalData("head", "state_id"), nil, nil),

		// Phase 3
		NewGenericFlattener("beacon_api_eth_v1_beacon_committee", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
		}, stringNotFromAdditionalData("finalized", "state_id"), nil, nil),
		NewGenericFlattener("canonical_beacon_committee", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
		}, stringFromAdditionalData("finalized", "state_id"), nil, nil),
		NewGenericFlattener("canonical_beacon_proposer_duty", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
		}, stringFromAdditionalData("finalized", "state_id"), nil, nil),
		NewGenericFlattener("beacon_api_eth_v2_beacon_block", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_block", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
		}, boolFromAdditionalData("finalized_when_requested"), nil, nil),
		NewGenericFlattener("beacon_api_eth_v3_validator_block", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V3_VALIDATOR_BLOCK,
		}, nil, nil, nil),
		NewValidatorsFanoutFlattener("canonical_beacon_validators", "validators"),
		NewValidatorsFanoutFlattener("canonical_beacon_validators_pubkeys", "pubkeys"),
		NewValidatorsFanoutFlattener("canonical_beacon_validators_withdrawal_credentials", "withdrawal_credentials"),
		NewGenericFlattener("canonical_beacon_block_attester_slashing", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_elaborated_attestation", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
		}, nil, nil, map[string]string{"validators": "validator_indexes"}),
		NewGenericFlattener("canonical_beacon_block_bls_to_execution_change", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_block_deposit", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_block_execution_transaction", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_block_proposer_slashing", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_block_voluntary_exit", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_block_withdrawal", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL,
		}, nil, nil, nil),
		NewGenericFlattener("canonical_beacon_sync_committee", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE,
		}, nil, syncCommitteeMutator, nil),
		NewGenericFlattener("canonical_beacon_block_sync_aggregate", []xatu.Event_Name{
			xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
		}, nil, syncAggregateMutator, nil),

		// Phase 4
		NewGenericFlattener("mev_relay_bid_trace", []xatu.Event_Name{
			xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
		}, nil, nil, map[string]string{"bid_trace_builder_block_submission": ""}),
		NewGenericFlattener("mev_relay_proposer_payload_delivered", []xatu.Event_Name{
			xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED,
		}, nil, nil, map[string]string{"payload_delivered": ""}),
		NewGenericFlattener("mev_relay_validator_registration", []xatu.Event_Name{
			xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
		}, nil, nil, nil),
		NewGenericFlattener("node_record_consensus", []xatu.Event_Name{
			xatu.Event_NODE_RECORD_CONSENSUS,
		}, nil, nil, nil),
		NewGenericFlattener("node_record_execution", []xatu.Event_Name{
			xatu.Event_NODE_RECORD_EXECUTION,
		}, nil, nil, nil),

		// Phase 5 - libp2p and convergence table
		NewGenericFlattener("libp2p_peer", []xatu.Event_Name{
			xatu.Event_LIBP2P_TRACE_CONNECTED,
			xatu.Event_LIBP2P_TRACE_DISCONNECTED,
			xatu.Event_LIBP2P_TRACE_ADD_PEER,
			xatu.Event_LIBP2P_TRACE_REMOVE_PEER,
			xatu.Event_LIBP2P_TRACE_RECV_RPC,
			xatu.Event_LIBP2P_TRACE_SEND_RPC,
			xatu.Event_LIBP2P_TRACE_DROP_RPC,
			xatu.Event_LIBP2P_TRACE_GRAFT,
			xatu.Event_LIBP2P_TRACE_PRUNE,
		}, nil, peerConvergenceMutator, nil),
		NewGenericFlattener("libp2p_connected", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_CONNECTED}, nil, nil, nil),
		NewGenericFlattener("libp2p_disconnected", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DISCONNECTED}, nil, nil, nil),
		NewGenericFlattener("libp2p_add_peer", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_ADD_PEER}, nil, nil, nil),
		NewGenericFlattener("libp2p_remove_peer", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_REMOVE_PEER}, nil, nil, nil),
		NewGenericFlattener("libp2p_recv_rpc", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RECV_RPC}, nil, nil, nil),
		NewGenericFlattener("libp2p_send_rpc", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SEND_RPC}, nil, nil, nil),
		NewGenericFlattener("libp2p_drop_rpc", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DROP_RPC}, nil, nil, nil),
		NewGenericFlattener("libp2p_join", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_JOIN}, nil, nil, nil),
		NewGenericFlattener("libp2p_leave", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_LEAVE}, nil, nil, nil),
		NewGenericFlattener("libp2p_graft", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GRAFT}, nil, nil, nil),
		NewGenericFlattener("libp2p_prune", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PRUNE}, nil, nil, nil),
		NewGenericFlattener("libp2p_publish_message", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE}, nil, nil, nil),
		NewGenericFlattener("libp2p_reject_message", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_REJECT_MESSAGE}, nil, nil, nil),
		NewGenericFlattener("libp2p_duplicate_message", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE}, nil, nil, nil),
		NewGenericFlattener("libp2p_deliver_message", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_DELIVER_MESSAGE}, nil, nil, nil),
		NewGenericFlattener("libp2p_handle_metadata", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_METADATA}, nil, nil, nil),
		NewGenericFlattener("libp2p_handle_status", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_HANDLE_STATUS}, nil, nil, nil),
		NewGenericFlattener("libp2p_gossipsub_beacon_block", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_BLOCK}, nil, nil, nil),
		NewGenericFlattener("libp2p_gossipsub_beacon_attestation", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION}, nil, nil, nil),
		NewGenericFlattener("libp2p_gossipsub_aggregate_and_proof", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF}, nil, nil, nil),
		NewGenericFlattener("libp2p_gossipsub_blob_sidecar", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR}, nil, nil, nil),
		NewGenericFlattener("libp2p_gossipsub_data_column_sidecar", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_data_column_custody_probe", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE}, nil, nil, nil),
		NewGenericFlattener("libp2p_synthetic_heartbeat", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_SYNTHETIC_HEARTBEAT}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_control_ihave", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IHAVE}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_control_iwant", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IWANT}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_control_idontwant", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_control_graft", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_GRAFT}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_control_prune", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_PRUNE}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_subscription", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_SUBSCRIPTION}, nil, nil, nil),
		NewGenericFlattener("libp2p_rpc_meta_message", []xatu.Event_Name{xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE}, nil, nil, nil),
	}

	return flatteners
}

func peerConvergenceMutator(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error) {
	peerID := firstNonEmpty(row, "remote_peer", "peer_id")
	if peerID == "" {
		return nil, nil
	}

	row["peer_id"] = peerID
	row["unique_key"] = hashKey(peerID + asString(row["meta_network_name"]))
	row["updated_date_time"] = time.Now().Unix()

	return []map[string]any{row}, nil
}

func syncCommitteeMutator(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error) {
	setAlias(row, "epoch", "epoch_number")

	aggsRaw, ok := row["sync_committee_validator_aggregates"]
	if !ok {
		return []map[string]any{row}, nil
	}

	aggs, ok := aggsRaw.([]any)
	if !ok {
		return []map[string]any{row}, nil
	}

	validatorAggregates := make([][]uint64, 0, len(aggs))

	for _, agg := range aggs {
		aggMap, ok := agg.(map[string]any)
		if !ok {
			continue
		}

		validators, _ := uint64Slice(aggMap["validators"])
		validatorAggregates = append(validatorAggregates, validators)
	}

	row["validator_aggregates"] = validatorAggregates

	return []map[string]any{row}, nil
}

func syncAggregateMutator(_ *xatu.DecoratedEvent, _ *metadata.CommonMetadata, row map[string]any) ([]map[string]any, error) {
	setAlias(row, "slot", "block_slot_number")
	setAlias(row, "epoch", "block_epoch_number")

	if validators, ok := uint64Slice(row["validators_participated"]); ok {
		row["validators_participated"] = validators
	}

	if validators, ok := uint64Slice(row["validators_missed"]); ok {
		row["validators_missed"] = validators
	}

	return []map[string]any{row}, nil
}

func uint64Slice(value any) ([]uint64, bool) {
	items, ok := value.([]any)
	if !ok {
		if values, okk := value.([]uint64); okk {
			return values, true
		}

		return nil, false
	}

	out := make([]uint64, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case uint64:
			out = append(out, v)
		case int64:
			if v >= 0 {
				out = append(out, uint64(v))
			}
		case float64:
			if v >= 0 {
				out = append(out, uint64(v))
			}
		case string:
			parsed, err := parseUint(v)
			if err == nil {
				out = append(out, parsed)
			}
		}
	}

	return out, true
}

func parseUint(value string) (uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("empty")
	}

	return strconv.ParseUint(value, 10, 64)
}
