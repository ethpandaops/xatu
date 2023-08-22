package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	ValidatorAttestationDataType = "BEACON_API_ETH_V1_VALIDATOR_ATTESTATION_DATA"
)

type ValidatorAttestationData struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewValidatorAttestationData(log logrus.FieldLogger, event *xatu.DecoratedEvent) *ValidatorAttestationData {
	return &ValidatorAttestationData{
		log:   log.WithField("event", ValidatorAttestationDataType),
		event: event,
	}
}

func (b *ValidatorAttestationData) Type() string {
	return ValidatorAttestationDataType
}

func (b *ValidatorAttestationData) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1ValidatorAttestationData)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *ValidatorAttestationData) Filter(_ context.Context) bool {
	return false
}
