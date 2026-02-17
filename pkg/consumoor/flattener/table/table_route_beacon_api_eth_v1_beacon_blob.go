package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1BeaconBlobRoute struct{}

func (BeaconApiEthV1BeaconBlobRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableBeaconApiEthV1BeaconBlob, xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB).
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
