package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcMetaMessageRoute struct{}

func (Libp2pRpcMetaMessageRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_rpc_meta_message")
}

func (r Libp2pRpcMetaMessageRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_RPC_META_MESSAGE).
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
	catalog.MustRegister(Libp2pRpcMetaMessageRoute{})
}
