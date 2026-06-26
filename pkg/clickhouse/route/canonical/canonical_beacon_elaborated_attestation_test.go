package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_elaborated_attestation(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconElaboratedAttestationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION,
			DateTime: testfixture.TS(),
			Id:       "cea-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockElaboratedAttestation{
				EthV2BeaconBlockElaboratedAttestation: &xatu.ClientMeta_AdditionalEthV2BeaconBlockElaboratedAttestationData{
					Block: &xatu.BlockIdentifier{
						Root:  "0x4444444444444444444444444444444444444444444444444444444444444444",
						Slot:  testfixture.SlotEpochAdditional(),
						Epoch: testfixture.EpochAdditional(),
					},
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
					Source: &xatu.ClientMeta_AdditionalEthV1AttestationSourceV2Data{
						Epoch: testfixture.EpochAdditional(),
					},
					Target: &xatu.ClientMeta_AdditionalEthV1AttestationTargetV2Data{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockElaboratedAttestation{
			EthV2BeaconBlockElaboratedAttestation: &ethv1.ElaboratedAttestation{
				ValidatorIndexes: []*wrapperspb.UInt64Value{
					wrapperspb.UInt64(11),
					wrapperspb.UInt64(22),
				},
				Data: &ethv1.AttestationDataV2{
					Slot:            wrapperspb.UInt64(100),
					Index:           wrapperspb.UInt64(0),
					BeaconBlockRoot: "0x1111111111111111111111111111111111111111111111111111111111111111",
					Source: &ethv1.CheckpointV2{
						Epoch: wrapperspb.UInt64(2),
						Root:  "0x2222222222222222222222222222222222222222222222222222222222222222",
					},
					Target: &ethv1.CheckpointV2{
						Epoch: wrapperspb.UInt64(3),
						Root:  "0x3333333333333333333333333333333333333333333333333333333333333333",
					},
				},
			},
		},
	}, 1, map[string]any{
		"meta_network_name": "mainnet",
		"beacon_block_root": "0x1111111111111111111111111111111111111111111111111111111111111111",
		"source_root":       "0x2222222222222222222222222222222222222222222222222222222222222222",
		"target_root":       "0x3333333333333333333333333333333333333333333333333333333333333333",
		"block_root":        "0x4444444444444444444444444444444444444444444444444444444444444444",
		"committee_index":   "0",
		"validators":        []uint32{11, 22},
	})
}
