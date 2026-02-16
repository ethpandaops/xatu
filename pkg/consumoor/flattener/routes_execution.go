package flattener

import "github.com/ethpandaops/xatu/pkg/proto/xatu"

func executionRoutes() []TableDefinition {
	return []TableDefinition{
		GenericTable(TableExecutionStateSize, []xatu.Event_Name{
			xatu.Event_EXECUTION_STATE_SIZE,
		}),
		GenericTable(TableMempoolTransaction, []xatu.Event_Name{
			xatu.Event_MEMPOOL_TRANSACTION,
			xatu.Event_MEMPOOL_TRANSACTION_V2,
		}),
		GenericTable(TableExecutionEngineNewPayload, []xatu.Event_Name{
			xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
		}),
		GenericTable(TableExecutionEngineGetBlobs, []xatu.Event_Name{
			xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
		}),
		GenericTable(TableExecutionBlockMetrics, []xatu.Event_Name{
			xatu.Event_EXECUTION_BLOCK_METRICS,
		}),
		GenericTable(TableConsensusEngineApiNewPayload, []xatu.Event_Name{
			xatu.Event_CONSENSUS_ENGINE_API_NEW_PAYLOAD,
		}),
		GenericTable(TableConsensusEngineApiGetBlobs, []xatu.Event_Name{
			xatu.Event_CONSENSUS_ENGINE_API_GET_BLOBS,
		}),
	}
}
