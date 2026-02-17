package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaControlIdontwantRoute struct{}

func (Libp2pRpcMetaControlIdontwantRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_rpc_meta_control_idontwant")
}

func (r Libp2pRpcMetaControlIdontwantRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_RPC_META_CONTROL_IDONTWANT).
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
	catalog.MustRegister(Libp2pRpcMetaControlIdontwantRoute{})
}
