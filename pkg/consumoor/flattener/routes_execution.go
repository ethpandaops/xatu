package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func executionRoutes() []Route {
	return []Route{
		RouteTo(TableExecutionStateSize, xatu.Event_EXECUTION_STATE_SIZE).
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
		RouteTo(
			TableMempoolTransaction,
			xatu.Event_MEMPOOL_TRANSACTION,
			xatu.Event_MEMPOOL_TRANSACTION_V2,
		).
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
		RouteTo(TableExecutionEngineNewPayload, xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD).
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
		RouteTo(TableExecutionEngineGetBlobs, xatu.Event_EXECUTION_ENGINE_GET_BLOBS).
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
		RouteTo(TableExecutionBlockMetrics, xatu.Event_EXECUTION_BLOCK_METRICS).
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
		RouteTo(TableConsensusEngineApiNewPayload, xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD).
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
		RouteTo(TableConsensusEngineApiGetBlobs, xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS).
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
