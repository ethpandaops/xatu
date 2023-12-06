package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsVoluntaryExitV2Type = "BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2"
)

type EventsVoluntaryExitV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsVoluntaryExitV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsVoluntaryExitV2 {
	return &EventsVoluntaryExitV2{
		log:   log.WithField("event", EventsVoluntaryExitV2Type),
		event: event,
	}
}

func (b *EventsVoluntaryExitV2) Type() string {
	return EventsVoluntaryExitV2Type
}

func (b *EventsVoluntaryExitV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsVoluntaryExitV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsVoluntaryExitV2) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsVoluntaryExitV2) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
