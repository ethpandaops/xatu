package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_beacon_api_eth_v1_events_proposer_preferences(t *testing.T) {
	if len(beaconApiEthV1EventsProposerPreferencesEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_proposer_preferences")
	}

	const (
		colValidatorIndex = "validator_index"
		colFeeRecipient   = "fee_recipient"
		colTargetGasLimit = "target_gas_limit"
		colEpoch          = "epoch"

		feeRecipient = "0x8943545177806ed17b9f23f0a21ee5948ecaa776"
	)

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsProposerPreferencesBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsProposerPreferencesEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsProposerPreferences{
				EthV1EventsProposerPreferences: &xatu.ClientMeta_AdditionalEthV1EventsProposerPreferencesData{
					Slot:        testfixture.SlotEpochAdditional(),
					Epoch:       testfixture.EpochAdditional(),
					Propagation: testfixture.PropagationAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsProposerPreferences{
			EthV1EventsProposerPreferences: &ethv1.SignedProposerPreferences{
				Message: &ethv1.ProposerPreferences{
					ProposalSlot:   wrapperspb.UInt64(48752),
					ValidatorIndex: wrapperspb.UInt64(1337),
					FeeRecipient:   feeRecipient,
					TargetGasLimit: wrapperspb.UInt64(60000000),
					DependentRoot:  repeatHex("4e", 32),
				},
				Signature: repeatHex("54", 96),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colValidatorIndex:             uint32(1337),
		colFeeRecipient:               feeRecipient,
		colTargetGasLimit:             uint64(60000000),
		colSlot:                       uint32(100),
		colEpoch:                      uint32(3),
	})
}
