package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsAttestationType = "BEACON_API_ETH_V1_EVENTS_ATTESTATION"
)

type EventsAttestation struct {
	log       logrus.FieldLogger
	event     *xatu.DecoratedEvent
	networkID uint64
}

func NewEventsAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent, networkID uint64) *EventsAttestation {
	return &EventsAttestation{
		log:       log.WithField("event", EventsAttestationType),
		event:     event,
		networkID: networkID,
	}
}

func (b *EventsAttestation) Type() string {
	return EventsAttestationType
}

func (b *EventsAttestation) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsAttestation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsAttestation) Filter(ctx context.Context) bool {
	networkID := b.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()

	return networkID != b.networkID
}
