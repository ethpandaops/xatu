package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsFinalizedCheckpointRoute struct{}

func (BeaconApiEthV1EventsFinalizedCheckpointRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsFinalizedCheckpoint,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT_V2,
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
