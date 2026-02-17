package table

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type CanonicalBeaconElaboratedAttestationRoute struct{}

func (CanonicalBeaconElaboratedAttestationRoute) Build() flattener.Route {
	return flattener.RouteTo(
		flattener.TableCanonicalBeaconElaboratedAttestation,
		xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
	).
		CommonMetadata().
		RuntimeColumns().
		EventData().
		ClientAdditionalData().
		ServerAdditionalData().
		TableAliases().
		RouteAliases().
		Aliases(map[string]string{"validator_indexes": "validators"}).
		NormalizeDateTimes().
		CommonEnrichment().
		Build()
}
