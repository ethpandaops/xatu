package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsAttestationV2Type = "BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2"
)

type EventsAttestationV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsAttestationV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsAttestationV2 {
	return &EventsAttestationV2{
		log:   log.WithField("event", EventsAttestationV2Type),
		event: event,
	}
}

func (b *EventsAttestationV2) Type() string {
	return EventsAttestationV2Type
}

func (b *EventsAttestationV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsAttestationV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsAttestationV2) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsAttestationV2) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
