package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_validators_pubkeys(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconValidatorsPubkeysBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
			DateTime: testfixture.TS(),
			Id:       "cvp-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1Validators{
				EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1Validators{
			EthV1Validators: &xatu.Validators{
				Validators: []*ethv1.Validator{
					{
						Index: wrapperspb.UInt64(42),
						Data: &ethv1.ValidatorData{
							Pubkey: wrapperspb.String("0xpub"),
						},
					},
				},
			},
		},
	}, 1, map[string]any{
		"index": uint32(42),
		"epoch": uint32(3),
	})
}
