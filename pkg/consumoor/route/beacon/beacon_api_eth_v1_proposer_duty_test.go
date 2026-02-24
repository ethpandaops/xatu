package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_proposer_duty(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1ProposerDutyBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_PROPOSER_DUTY,
			DateTime: testfixture.TS(),
			Id:       "proposer-duty-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1ProposerDuty{
				EthV1ProposerDuty: &xatu.ClientMeta_AdditionalEthV1ProposerDutyData{
					StateId: "head",
					Epoch:   testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1ProposerDuty{
			EthV1ProposerDuty: &ethv1.ProposerDuty{
				Slot:           wrapperspb.UInt64(100),
				ValidatorIndex: wrapperspb.UInt64(55),
				Pubkey:         "0xpub",
			},
		},
	}, 1, map[string]any{
		"slot":                     uint32(100),
		"proposer_validator_index": uint32(55),
		"meta_client_name":         "test-client",
		"meta_network_name":        "mainnet",
	})
}
