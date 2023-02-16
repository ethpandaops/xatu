package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsHeadType = "BEACON_API_ETH_V1_EVENTS_HEAD"
)

type EventsHead struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsHead(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsHead {
	return &EventsHead{
		log:   log.WithField("event", EventsHeadType),
		event: event,
	}
}

func (b *EventsHead) Type() string {
	return EventsHeadType
}

func (b *EventsHead) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_EthV1EventsHead)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsHead) Filter(ctx context.Context) bool {
	return false
}
