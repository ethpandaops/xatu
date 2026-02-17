package execution

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type MempoolTransactionRoute struct{}

func (MempoolTransactionRoute) Table() flattener.TableName {
	return flattener.TableName("mempool_transaction")
}

func (r MempoolTransactionRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_MEMPOOL_TRANSACTION,
			xatu.Event_MEMPOOL_TRANSACTION_V2).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.CopyFieldIfMissing("raw", "mempool_transaction")).
		Apply(flattener.CopyFieldIfMissing("raw", "mempool_transaction_v2")).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(MempoolTransactionRoute{})
}
