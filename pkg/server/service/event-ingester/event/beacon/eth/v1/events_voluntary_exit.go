package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	EventsVoluntaryExitType = "BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT"
)

type EventsVoluntaryExit struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewEventsVoluntaryExit(log observability.ContextualLogger, event *xatu.DecoratedEvent) *EventsVoluntaryExit {
	return &EventsVoluntaryExit{
		log:   log.WithField("event", EventsVoluntaryExitType),
		event: event,
	}
}

func (b *EventsVoluntaryExit) Type() string {
	return EventsVoluntaryExitType
}

func (b *EventsVoluntaryExit) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsVoluntaryExit)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsVoluntaryExit) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsVoluntaryExit) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
