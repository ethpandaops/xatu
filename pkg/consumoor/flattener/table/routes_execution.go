package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

func executionRoutes() []flattener.Route {
	return []flattener.Route{
		ExecutionStateSizeRoute{}.Build(),
		MempoolTransactionRoute{}.Build(),
		ExecutionEngineNewPayloadRoute{}.Build(),
		ExecutionEngineGetBlobsRoute{}.Build(),
		ExecutionBlockMetricsRoute{}.Build(),
		ConsensusEngineApiNewPayloadRoute{}.Build(),
		ConsensusEngineApiGetBlobsRoute{}.Build(),
	}
}
