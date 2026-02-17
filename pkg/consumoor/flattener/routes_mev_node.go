package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func mevAndNodeRoutes() []Route {
	return []Route{
		RouteTo(
			TableMevRelayBidTrace,
			xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION,
		).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			Aliases(map[string]string{"bid_trace_builder_block_submission": ""}).
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(
			TableMevRelayProposerPayloadDelivered,
			xatu.Event_MEV_RELAY_PROPOSER_PAYLOAD_DELIVERED,
		).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			Aliases(map[string]string{"payload_delivered": ""}).
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableMevRelayValidatorRegistration, xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableNodeRecordConsensus, xatu.Event_NODE_RECORD_CONSENSUS).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
		RouteTo(TableNodeRecordExecution, xatu.Event_NODE_RECORD_EXECUTION).
			CommonMetadata().
			RuntimeColumns().
			EventData().
			ClientAdditionalData().
			ServerAdditionalData().
			TableAliases().
			RouteAliases().
			NormalizeDateTimes().
			CommonEnrichment().
			Build(),
	}
}
