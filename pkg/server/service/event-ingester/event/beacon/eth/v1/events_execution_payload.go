package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	EventsExecutionPayloadType = "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD"
)

type EventsExecutionPayload struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewEventsExecutionPayload(log observability.ContextualLogger, event *xatu.DecoratedEvent) *EventsExecutionPayload {
	return &EventsExecutionPayload{
		log:   log.WithField("event", EventsExecutionPayloadType),
		event: event,
	}
}

func (e *EventsExecutionPayload) Type() string {
	return EventsExecutionPayloadType
}

func (e *EventsExecutionPayload) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsExecutionPayload)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *EventsExecutionPayload) Filter(_ context.Context) bool {
	return false
}

func (e *EventsExecutionPayload) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
