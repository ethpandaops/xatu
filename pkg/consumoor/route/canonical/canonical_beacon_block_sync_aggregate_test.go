package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_sync_aggregate(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockSyncAggregateBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE,
			DateTime: testfixture.TS(),
			Id:       "cbsa-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockSyncAggregate{
			EthV2BeaconBlockSyncAggregate: &xatu.SyncAggregateData{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
