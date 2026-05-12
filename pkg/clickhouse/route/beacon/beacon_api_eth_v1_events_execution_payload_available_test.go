package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_execution_payload_available(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadAvailableEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload_available")
	}

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadAvailableBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadAvailableEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.BaseMeta(),
		// TODO: Add event-specific Data field and MetaWithAdditional for richer assertions.
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		// TODO: Add payload-specific column assertions.
	})
}
