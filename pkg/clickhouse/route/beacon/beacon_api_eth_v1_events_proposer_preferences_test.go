package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_proposer_preferences(t *testing.T) {
	if len(beaconApiEthV1EventsProposerPreferencesEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_proposer_preferences")
	}

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsProposerPreferencesBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsProposerPreferencesEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.BaseMeta(),
		// TODO(epbs): Add event-specific Data field and MetaWithAdditional for richer assertions.
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		// TODO(epbs): Add payload-specific column assertions.
	})
}
