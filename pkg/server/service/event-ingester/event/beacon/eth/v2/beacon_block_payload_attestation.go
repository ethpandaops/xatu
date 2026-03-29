package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockPayloadAttestationType = "BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION"
)

type BeaconBlockPayloadAttestation struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockPayloadAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockPayloadAttestation {
	return &BeaconBlockPayloadAttestation{
		log:   log.WithField("event", BeaconBlockPayloadAttestationType),
		event: event,
	}
}

func (b *BeaconBlockPayloadAttestation) Type() string {
	return BeaconBlockPayloadAttestationType
}

func (b *BeaconBlockPayloadAttestation) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV2BeaconBlockPayloadAttestation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockPayloadAttestation) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconBlockPayloadAttestation) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
