package libp2p

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubBlobSidecarRoute struct{}

func (Libp2pGossipsubBlobSidecarRoute) Table() flattener.TableName {
	return flattener.TableName("libp2p_gossipsub_blob_sidecar")
}

func (r Libp2pGossipsubBlobSidecarRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR).
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
	catalog.MustRegister(Libp2pGossipsubBlobSidecarRoute{})
}
