package mevrelay

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	ValidatorRegistrationType = xatu.Event_MEV_RELAY_VALIDATOR_REGISTRATION.String()
)

type ValidatorRegistration struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewValidatorRegistration(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ValidatorRegistration {
	return &ValidatorRegistration{
		log:   log.WithField("event", ValidatorRegistrationType),
		event: event,
	}
}

func (b *ValidatorRegistration) Type() string {
	return ValidatorRegistrationType
}

func (b *ValidatorRegistration) Validate(ctx context.Context) error {
	ev, ok := b.event.Data.(*xatu.DecoratedEvent_MevRelayValidatorRegistration)
	if !ok {
		return errors.New("failed to cast event data")
	}

	if ev.MevRelayValidatorRegistration.GetMessage().GetPubkey().String() == "" {
		return errors.New("validator pubkey is empty")
	}

	return nil
}

func (b *ValidatorRegistration) Filter(ctx context.Context) bool {
	return false
}

func (b *ValidatorRegistration) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
