package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func mevAndNodeRoutes() []Flattener {
	return []Flattener{
		NewGenericRoute(
			TableMevRelayBidTrace,
			[]xatu.Event_Name{xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION},
			WithAliases(map[string]string{"bid_trace_builder_block_submission": ""}),
		),
		NewGenericRoute(
			TableMevRelayProposerPayloadDelivered,
			[]xatu.Event_Name{xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED},
			WithAliases(map[string]string{"payload_delivered": ""}),
		),
		NewGenericRoute(TableMevRelayValidatorRegistration, []xatu.Event_Name{
			xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
		}),
		NewGenericRoute(TableNodeRecordConsensus, []xatu.Event_Name{
			xatu.Event_NODE_RECORD_CONSENSUS,
		}),
		NewGenericRoute(TableNodeRecordExecution, []xatu.Event_Name{
			xatu.Event_NODE_RECORD_EXECUTION,
		}),
	}
}
