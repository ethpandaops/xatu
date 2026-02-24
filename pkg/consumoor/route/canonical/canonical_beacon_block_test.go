package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_block(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_V2,
			DateTime: testfixture.TS(),
			Id:       "cbb-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockV2{
				EthV2BeaconBlockV2: &xatu.ClientMeta_AdditionalEthV2BeaconBlockV2Data{
					FinalizedWhenRequested: true,
					Slot:                   testfixture.SlotEpochAdditional(),
					Epoch:                  testfixture.EpochAdditional(),
					Version:                "deneb",
					BlockRoot:              "0xblockroot",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockV2{
			EthV2BeaconBlockV2: &ethv2.EventBlockV2{
				Message: &ethv2.EventBlockV2_DenebBlock{
					DenebBlock: &ethv2.BeaconBlockDeneb{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
					},
				},
			},
		},
	}, 1, map[string]any{
		"block_version": "deneb",
		"block_root":    "0xblockroot",
		"slot":          uint32(100),
	})
}
