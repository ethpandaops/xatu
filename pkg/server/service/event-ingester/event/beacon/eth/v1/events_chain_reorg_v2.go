package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsChainReorgV2Type = "BEACON_API_ETH_V1_EVENTS_CHAIN_REORG_V2"
)

type EventsChainReorgV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsChainReorgV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsChainReorgV2 {
	return &EventsChainReorgV2{
		log:   log.WithField("event", EventsChainReorgV2Type),
		event: event,
	}
}

func (b *EventsChainReorgV2) Type() string {
	return EventsChainReorgV2Type
}

func (b *EventsChainReorgV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsChainReorgV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsChainReorgV2) Filter(ctx context.Context) bool {
	return false
}
