package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MempoolTransactionRoute struct{}

func (MempoolTransactionRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableMempoolTransaction,
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
		Build()
}
