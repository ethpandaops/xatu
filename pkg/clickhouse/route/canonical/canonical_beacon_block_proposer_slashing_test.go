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
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &ethv1.ProposerSlashingV2{
				SignedHeader_1: &ethv1.SignedBeaconBlockHeaderV2{
					Message: &ethv1.BeaconBlockHeaderV2{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
					},
				},
				SignedHeader_2: &ethv1.SignedBeaconBlockHeaderV2{
					Message: &ethv1.BeaconBlockHeaderV2{
						Slot:          wrapperspb.UInt64(100),
						ProposerIndex: wrapperspb.UInt64(42),
					},
				},
			},
		},
	}, 1, map[string]any{
		"meta_network_name": "mainnet",
	})
}
