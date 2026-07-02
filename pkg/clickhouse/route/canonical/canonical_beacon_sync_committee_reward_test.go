package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_sync_committee_reward(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconSyncCommitteeRewardBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD,
			DateTime: testfixture.TS(),
			Id:       "scr-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconSyncCommitteeReward{
				EthV1BeaconSyncCommitteeReward: &xatu.ClientMeta_AdditionalEthV1BeaconSyncCommitteeRewardData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
						Version: "deneb",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconSyncCommitteeReward{
			EthV1BeaconSyncCommitteeReward: &xatu.SyncCommitteeRewardData{
				ValidatorIndex: wrapperspb.UInt64(42),
				Reward:         wrapperspb.Int64(-1234),
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"epoch":             uint32(3),
		"validator_index":   uint32(42),
		"reward":            int64(-1234),
		"block_root":        "0x1111111111111111111111111111111111111111111111111111111111111111",
		"meta_network_name": "mainnet",
	})
}
