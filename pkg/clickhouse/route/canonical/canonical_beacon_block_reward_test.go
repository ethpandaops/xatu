package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_reward(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockRewardBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOCK_REWARD,
			DateTime: testfixture.TS(),
			Id:       "cbr-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconBlockReward{
				EthV1BeaconBlockReward: &xatu.ClientMeta_AdditionalEthV1BeaconBlockRewardData{
					Block: &xatu.BlockIdentifier{
						Slot:  testfixture.SlotEpochAdditional(),
						Epoch: testfixture.EpochAdditional(),
						Root:  "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconBlockReward{
			EthV1BeaconBlockReward: &xatu.BlockRewardData{
				ProposerIndex:     wrapperspb.UInt64(42),
				Total:             wrapperspb.UInt64(1000),
				Attestations:      wrapperspb.UInt64(700),
				SyncAggregate:     wrapperspb.UInt64(250),
				ProposerSlashings: wrapperspb.UInt64(30),
				AttesterSlashings: wrapperspb.UInt64(20),
			},
		},
	}, 1, map[string]any{
		"meta_network_name":  "mainnet",
		"slot":               uint32(100),
		"epoch":              uint32(3),
		"proposer_index":     uint32(42),
		"total":              uint64(1000),
		"attestations":       uint64(700),
		"sync_aggregate":     uint64(250),
		"proposer_slashings": uint64(30),
		"attester_slashings": uint64(20),
		"block_root":         "0x0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20",
	})
}
