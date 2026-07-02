package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_proposer_slashing(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockProposerSlashingBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING,
			DateTime: testfixture.TS(),
			Id:       "cps-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockProposerSlashing{
				EthV2BeaconBlockProposerSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockProposerSlashingData{
					Block: &xatu.BlockIdentifier{
						Epoch:   testfixture.EpochAdditional(),
						Slot:    testfixture.SlotEpochAdditional(),
						Version: "deneb",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &ethv1.ProposerSlashingV2{
				SignedHeader_1: &ethv1.SignedBeaconBlockHeaderV2{
					Signature: "0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9011",
					Message: &ethv1.BeaconBlockHeaderV2{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
						ParentRoot:    "0x2222222222222222222222222222222222222222222222222222222222222222",
						StateRoot:     "0x3333333333333333333333333333333333333333333333333333333333333333",
						BodyRoot:      "0x4444444444444444444444444444444444444444444444444444444444444444",
					},
				},
				SignedHeader_2: &ethv1.SignedBeaconBlockHeaderV2{
					Signature: "0xb1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9022",
					Message: &ethv1.BeaconBlockHeaderV2{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
						ParentRoot:    "0x5555555555555555555555555555555555555555555555555555555555555555",
						StateRoot:     "0x6666666666666666666666666666666666666666666666666666666666666666",
						BodyRoot:      "0x7777777777777777777777777777777777777777777777777777777777777777",
					},
				},
			},
		},
	}, 1, map[string]any{
		"meta_network_name":                  "mainnet",
		"block_root":                         "0x1111111111111111111111111111111111111111111111111111111111111111",
		"block_version":                      "deneb",
		"signed_header_1_signature":          "0xa1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9011",
		"signed_header_1_message_body_root":  "0x4444444444444444444444444444444444444444444444444444444444444444",
		"signed_header_2_message_state_root": "0x6666666666666666666666666666666666666666666666666666666666666666",
	})
}
