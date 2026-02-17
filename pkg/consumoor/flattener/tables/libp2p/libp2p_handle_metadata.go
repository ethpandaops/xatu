package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pHandleMetadataRoute struct{}

func (Libp2pHandleMetadataRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_handle_metadata")
}

func (r Libp2pHandleMetadataRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_HANDLE_METADATA).
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
	catalog.MustRegister(Libp2pHandleMetadataRoute{})
}
