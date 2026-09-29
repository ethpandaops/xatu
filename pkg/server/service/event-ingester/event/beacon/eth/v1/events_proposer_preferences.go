package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	EventsProposerPreferencesType = "BEACON_API_ETH_V1_EVENTS_PROPOSER_PREFERENCES"
)

type EventsProposerPreferences struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewEventsProposerPreferences(log observability.ContextualLogger, event *xatu.DecoratedEvent) *EventsProposerPreferences {
	return &EventsProposerPreferences{
		log:   log.WithField("event", EventsProposerPreferencesType),
		event: event,
	}
}

func (e *EventsProposerPreferences) Type() string {
	return EventsProposerPreferencesType
}

func (e *EventsProposerPreferences) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsProposerPreferences)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *EventsProposerPreferences) Filter(_ context.Context) bool {
	return false
}

func (e *EventsProposerPreferences) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
