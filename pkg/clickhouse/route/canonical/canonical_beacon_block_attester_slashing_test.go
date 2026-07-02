package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

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
						Slot:    testfixture.SlotEpochAdditional(),
						Epoch:   testfixture.EpochAdditional(),
						Version: "deneb",
						Root:    "0x1111111111111111111111111111111111111111111111111111111111111111",
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &ethv1.AttesterSlashingV2{
				Attestation_1: &ethv1.IndexedAttestationV2{
					Signature: "0xaaaa",
					Data: &ethv1.AttestationDataV2{
						BeaconBlockRoot: "0x2222222222222222222222222222222222222222222222222222222222222222",
					},
				},
				Attestation_2: &ethv1.IndexedAttestationV2{
					Signature: "0xbbbb",
					Data: &ethv1.AttestationDataV2{
						BeaconBlockRoot: "0x3333333333333333333333333333333333333333333333333333333333333333",
					},
				},
			},
		},
	}, 1, map[string]any{
		"meta_network_name": "mainnet",
	})
}
