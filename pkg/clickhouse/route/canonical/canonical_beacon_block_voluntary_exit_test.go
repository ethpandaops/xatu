package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_voluntary_exit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockVoluntaryExitBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			DateTime: testfixture.TS(),
			Id:       "cbve-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
				EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
					Block: &xatu.BlockIdentifier{
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "deneb",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &ethv1.SignedVoluntaryExitV2{
				Message: &ethv1.VoluntaryExitV2{
					Epoch:          wrapperspb.UInt64(3),
					ValidatorIndex: wrapperspb.UInt64(42),
				},
				Signature: "0x" + "ab" + "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			},
		},
	}, 1, map[string]any{
		"meta_network_name":                      "mainnet",
		"block_version":                          "deneb",
		"voluntary_exit_message_epoch":           uint32(3),
		"voluntary_exit_message_validator_index": uint32(42),
	})
}
