package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pDuplicateMessageRoute struct{}

func (Libp2pDuplicateMessageRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_duplicate_message")
}

func (r Libp2pDuplicateMessageRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_DUPLICATE_MESSAGE).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Apply(EnrichFields).
		Build()
}

func init() {
	catalog.MustRegister(Libp2pDuplicateMessageRoute{})
}
