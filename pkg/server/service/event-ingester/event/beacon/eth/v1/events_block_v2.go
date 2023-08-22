package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsBlockV2Type = "BEACON_API_ETH_V1_EVENTS_BLOCK_V2"
)

type EventsBlockV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsBlockV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsBlockV2 {
	return &EventsBlockV2{
		log:   log.WithField("event", EventsBlockV2Type),
		event: event,
	}
}

func (b *EventsBlockV2) Type() string {
	return EventsBlockV2Type
}

func (b *EventsBlockV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsBlockV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsBlockV2) Filter(ctx context.Context) bool {
	return false
}
