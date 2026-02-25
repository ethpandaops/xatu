package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_validator_attestation_data(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1ValidatorAttestationDataBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA,
			DateTime: testfixture.TS(),
			Id:       "validator-attestation-data-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1ValidatorAttestationData{
				EthV1ValidatorAttestationData: &xatu.ClientMeta_AdditionalEthV1ValidatorAttestationDataData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1ValidatorAttestationData{
			EthV1ValidatorAttestationData: &ethv1.AttestationDataV2{
				Slot:  wrapperspb.UInt64(100),
				Index: wrapperspb.UInt64(8),
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"committee_index":   "8",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
