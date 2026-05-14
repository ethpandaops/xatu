package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
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
			EthV2BeaconBlockSyncAggregate: &xatu.SyncAggregateData{
				ParticipationCount: wrapperspb.UInt64(128),
			},
		},
	}, 1, map[string]any{
		"meta_network_name": "mainnet",
	})
}
