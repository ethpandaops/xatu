package beacon

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsChainReorgRoute struct{}

func (BeaconApiEthV1EventsChainReorgRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_events_chain_reorg")
}

func (r BeaconApiEthV1EventsChainReorgRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG,
			xatu.Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2).
		To(r.Table()).
		Apply(flattener.AddCommonMetadataFields).
		Apply(flattener.AddRuntimeColumns).
		Apply(flattener.FlattenEventDataFields).
		Apply(flattener.FlattenClientAdditionalDataFields).
		Apply(flattener.FlattenServerAdditionalDataFields).
		Apply(flattener.NormalizeDateTimeValues).
		Build()
}

func init() {
	catalog.MustRegister(BeaconApiEthV1EventsChainReorgRoute{})
}
