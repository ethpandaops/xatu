package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsExecutionPayloadAvailableType = "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_AVAILABLE"
)

type EventsExecutionPayloadAvailable struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsExecutionPayloadAvailable(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsExecutionPayloadAvailable {
	return &EventsExecutionPayloadAvailable{
		log:   log.WithField("event", EventsExecutionPayloadAvailableType),
		event: event,
	}
}

func (e *EventsExecutionPayloadAvailable) Type() string {
	return EventsExecutionPayloadAvailableType
}

func (e *EventsExecutionPayloadAvailable) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsExecutionPayloadAvailable)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *EventsExecutionPayloadAvailable) Filter(_ context.Context) bool {
	return false
}

func (e *EventsExecutionPayloadAvailable) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
