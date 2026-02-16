package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func executionRoutes() []Flattener {
	return []Flattener{
		NewGenericRoute(TableExecutionStateSize, []xatu.Event_Name{
			xatu.Event_EXECUTION_STATE_SIZE,
		}),
		NewGenericRoute(TableMempoolTransaction, []xatu.Event_Name{
			xatu.Event_MEMPOOL_TRANSACTION,
			xatu.Event_MEMPOOL_TRANSACTION_V2,
		}),
		NewGenericRoute(TableExecutionEngineNewPayload, []xatu.Event_Name{
			xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
		}),
		NewGenericRoute(TableExecutionEngineGetBlobs, []xatu.Event_Name{
			xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
		}),
		NewGenericRoute(TableExecutionBlockMetrics, []xatu.Event_Name{
			xatu.Event_EXECUTION_BLOCK_METRICS,
		}),
		NewGenericRoute(TableConsensusEngineApiNewPayload, []xatu.Event_Name{
			xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
		}),
		NewGenericRoute(TableConsensusEngineApiGetBlobs, []xatu.Event_Name{
			xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS,
		}),
	}
}
