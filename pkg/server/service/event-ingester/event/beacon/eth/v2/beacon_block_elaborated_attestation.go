package v2

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconBlockElaboratedAttestationType = "BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION"
)

type BeaconBlockElaboratedAttestation struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBeaconBlockElaboratedAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BeaconBlockElaboratedAttestation {
	return &BeaconBlockElaboratedAttestation{
		log:   log.WithField("event", BeaconBlockElaboratedAttestationType),
		event: event,
	}
}

func (b *BeaconBlockElaboratedAttestation) Type() string {
	return BeaconBlockElaboratedAttestationType
}

func (b *BeaconBlockElaboratedAttestation) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV2BeaconBlockElaboratedAttestation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconBlockElaboratedAttestation) Filter(ctx context.Context) bool {
	return false
}

func (b *BeaconBlockElaboratedAttestation) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
