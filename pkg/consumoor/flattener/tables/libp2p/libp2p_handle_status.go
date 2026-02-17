package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pHandleStatusRoute struct{}

func (Libp2pHandleStatusRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_handle_status")
}

func (r Libp2pHandleStatusRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_HANDLE_STATUS).
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
	catalog.MustRegister(Libp2pHandleStatusRoute{})
}
