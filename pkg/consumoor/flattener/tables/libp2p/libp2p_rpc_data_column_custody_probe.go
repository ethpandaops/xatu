package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pRpcDataColumnCustodyProbeRoute struct{}

func (Libp2pRpcDataColumnCustodyProbeRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_rpc_data_column_custody_probe")
}

func (r Libp2pRpcDataColumnCustodyProbeRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_RPC_DATA_COLUMN_CUSTODY_PROBE).
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
	catalog.MustRegister(Libp2pRpcDataColumnCustodyProbeRoute{})
}
