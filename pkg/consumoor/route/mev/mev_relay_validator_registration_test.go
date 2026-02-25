package mev

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	mevrelay "github.com/ethpandaops/xatu/pkg/proto/mevrelay"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_mev_relay_validator_registration(t *testing.T) {
	testfixture.AssertSnapshot(t, newmevRelayValidatorRegistrationBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION,
			DateTime: testfixture.TS(),
			Id:       "mevvr-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_MevRelayValidatorRegistration{
				MevRelayValidatorRegistration: &xatu.ClientMeta_AdditionalMevRelayValidatorRegistrationData{
					ValidatorIndex: wrapperspb.UInt64(42),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_MevRelayValidatorRegistration{
			MevRelayValidatorRegistration: &mevrelay.ValidatorRegistration{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
