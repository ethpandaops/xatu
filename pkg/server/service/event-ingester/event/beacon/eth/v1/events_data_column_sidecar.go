package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	EventsDataColumnSidecarType = "BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR"
)

type EventsDataColumnSidecar struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewEventsDataColumnSidecar(log observability.ContextualLogger, event *xatu.DecoratedEvent) *EventsDataColumnSidecar {
	return &EventsDataColumnSidecar{
		log:   log.WithField("event", EventsDataColumnSidecarType),
		event: event,
	}
}

func (e *EventsDataColumnSidecar) Type() string {
	return EventsDataColumnSidecarType
}

func (e *EventsDataColumnSidecar) Validate(ctx context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsDataColumnSidecar)
	if !ok {
		return errors.New("failed to cast event data to data column sidecar event")
	}

	return nil
}

func (e *EventsDataColumnSidecar) Filter(ctx context.Context) bool {
	return false
}

func (e *EventsDataColumnSidecar) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
