package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Libp2pGossipsubBeaconAttestationRoute struct{}

func (Libp2pGossipsubBeaconAttestationRoute) Build() flattener.Route {
	return flattener.RouteTo(flattener.TableLibp2pGossipsubBeaconAttestation, xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION).
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
