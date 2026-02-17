package beacon

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	catalog "github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables/catalog"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type BeaconApiEthV1EventsBlockGossipRoute struct{}

func (BeaconApiEthV1EventsBlockGossipRoute) Table() flattener.TableName {
	return flattener.TableName("beacon_api_eth_v1_events_block_gossip")
}

func (r BeaconApiEthV1EventsBlockGossipRoute) Build() flattener.Route {
	return flattener.
		From(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP).
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
	catalog.MustRegister(BeaconApiEthV1EventsBlockGossipRoute{})
}
