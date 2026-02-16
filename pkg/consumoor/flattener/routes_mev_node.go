package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func mevAndNodeRoutes() []TableDefinition {
	return []TableDefinition{
		GenericTable(
			TableMevRelayBidTrace,
			[]xatu.Event_Name{xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION},
			WithAliases(map[string]string{"bid_trace_builder_block_submission": ""}),
		),
		GenericTable(
			TableMevRelayProposerPayloadDelivered,
			[]xatu.Event_Name{xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED},
			WithAliases(map[string]string{"payload_delivered": ""}),
		),
		GenericTable(TableMevRelayValidatorRegistration, []xatu.Event_Name{
			xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
		}),
		GenericTable(TableNodeRecordConsensus, []xatu.Event_Name{
			xatu.Event_NODE_RECORD_CONSENSUS,
		}),
		GenericTable(TableNodeRecordExecution, []xatu.Event_Name{
			xatu.Event_NODE_RECORD_EXECUTION,
		}),
	}
}
