package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsPayloadAttestationType = "BEACON_API_ETH_V1_EVENTS_PAYLOAD_ATTESTATION"
)

type EventsPayloadAttestation struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsPayloadAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsPayloadAttestation {
	return &EventsPayloadAttestation{
		log:   log.WithField("event", EventsPayloadAttestationType),
		event: event,
	}
}

func (e *EventsPayloadAttestation) Type() string {
	return EventsPayloadAttestationType
}

func (e *EventsPayloadAttestation) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsPayloadAttestation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *EventsPayloadAttestation) Filter(_ context.Context) bool {
	return false
}

func (e *EventsPayloadAttestation) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
