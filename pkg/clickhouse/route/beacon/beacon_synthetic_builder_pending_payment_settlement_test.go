package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_synthetic_builder_pending_payment_settlement(t *testing.T) {
	if len(beaconSyntheticBuilderPendingPaymentSettlementEventNames) == 0 {
		t.Skip("no event names registered for beacon_synthetic_builder_pending_payment_settlement")
	}

	testfixture.AssertSnapshot(t, newbeaconSyntheticBuilderPendingPaymentSettlementBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconSyntheticBuilderPendingPaymentSettlementEventNames[0],
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
