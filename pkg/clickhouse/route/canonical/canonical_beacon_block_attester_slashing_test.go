package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func indexedAttestationFixture() *ethv1.IndexedAttestationV2 {
	return &ethv1.IndexedAttestationV2{
		AttestingIndices: []*wrapperspb.UInt64Value{wrapperspb.UInt64(1), wrapperspb.UInt64(2)},
		Data: &ethv1.AttestationDataV2{
			Slot:   wrapperspb.UInt64(100),
			Index:  wrapperspb.UInt64(3),
			Source: &ethv1.CheckpointV2{Epoch: wrapperspb.UInt64(4)},
			Target: &ethv1.CheckpointV2{Epoch: wrapperspb.UInt64(5)},
		},
	}
}

func TestSnapshot_canonical_beacon_block_attester_slashing(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockAttesterSlashingBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING,
			DateTime: testfixture.TS(),
			Id:       "cas-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockAttesterSlashing{
				EthV2BeaconBlockAttesterSlashing: &xatu.ClientMeta_AdditionalEthV2BeaconBlockAttesterSlashingData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &ethv1.AttesterSlashingV2{
				Attestation_1: indexedAttestationFixture(),
				Attestation_2: indexedAttestationFixture(),
			},
		},
	}, 1, map[string]any{
		"meta_network_name": "mainnet",
	})
}
