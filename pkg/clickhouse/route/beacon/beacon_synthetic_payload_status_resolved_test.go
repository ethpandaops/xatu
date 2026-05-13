package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_synthetic_payload_status_resolved(t *testing.T) {
	if len(beaconSyntheticPayloadStatusResolvedEventNames) == 0 {
		t.Skip("no event names registered for beacon_synthetic_payload_status_resolved")
	}

	testfixture.AssertSnapshot(t, newbeaconSyntheticPayloadStatusResolvedBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconSyntheticPayloadStatusResolvedEventNames[0],
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
