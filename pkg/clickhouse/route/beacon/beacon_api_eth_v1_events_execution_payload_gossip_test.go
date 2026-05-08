package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_execution_payload_gossip(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadGossipEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload_gossip")
	}

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadGossipBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadGossipEventNames[0],
			DateTime: testfixture.TS(),
			Id:       "snapshot-1",
		},
		Meta: testfixture.BaseMeta(),
		// TODO: Add event-specific Data field and MetaWithAdditional for richer assertions.
	}, 1, map[string]any{
		"meta_client_name": "test-client",
		// TODO: Add payload-specific column assertions.
	})
}
