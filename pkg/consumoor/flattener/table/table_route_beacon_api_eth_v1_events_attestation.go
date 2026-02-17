package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsAttestationRoute struct{}

func (BeaconApiEthV1EventsAttestationRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableBeaconApiEthV1EventsAttestation,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION,
		xatu.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2,
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
