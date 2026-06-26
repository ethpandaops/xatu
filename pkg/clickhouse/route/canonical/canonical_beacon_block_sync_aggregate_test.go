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
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockSyncAggregate{
				EthV2BeaconBlockSyncAggregate: &xatu.ClientMeta_AdditionalEthV2BeaconBlockSyncAggregateData{
					Block: &xatu.BlockIdentifier{
						Version: "deneb",
						Root:    "0xdeadbeef000000000000000000000000000000000000000000000000deadbeef",
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockSyncAggregate{
			EthV2BeaconBlockSyncAggregate: &xatu.SyncAggregateData{
				ParticipationCount:     wrapperspb.UInt64(128),
				SyncCommitteeBits:      "0xffffffffffffffffffffffffffffffff",
				SyncCommitteeSignature: "0xb0b0b0b0",
			},
		},
	}, 1, map[string]any{
		"meta_network_name": "mainnet",
		"block_version":     "deneb",
	})
}
