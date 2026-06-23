package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_execution_request_deposit(t *testing.T) {
	if len(canonicalBeaconBlockExecutionRequestDepositEventNames) == 0 {
		t.Skip("no event names registered for canonical_beacon_block_execution_request_deposit")
	}

	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockExecutionRequestDepositBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     canonicalBeaconBlockExecutionRequestDepositEventNames[0],
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
