package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconBlobSidecarRoute struct{}

func (CanonicalBeaconBlobSidecarRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconBlobSidecar,
		xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
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
