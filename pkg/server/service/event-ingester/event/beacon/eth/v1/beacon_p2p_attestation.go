package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconP2PAttestationType = "BEACON_P2P_ATTESTATION"
)

type BeaconP2PAttestation struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconP2PAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconP2PAttestation {
	return &BeaconP2PAttestation{
		log:   log.WithField("event", BeaconP2PAttestationType),
		event: event,
	}
}

func (b *BeaconP2PAttestation) Type() string {
	return BeaconP2PAttestationType
}

func (b *BeaconP2PAttestation) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_BeaconP2PAttestation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconP2PAttestation) Filter(_ context.Context) bool {
	return false
}
