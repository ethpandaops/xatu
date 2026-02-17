package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsDataColumnSidecarRoute struct{}

func (BeaconApiEthV1EventsDataColumnSidecarRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsDataColumnSidecar,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
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
