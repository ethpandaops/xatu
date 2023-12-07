package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsChainReorgType = "BEACON_API_ETH_V1_EVENTS_CHAIN_REORG"
)

type EventsChainReorg struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsChainReorg(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsChainReorg {
	return &EventsChainReorg{
		log:   log.WithField("event", EventsChainReorgType),
		event: event,
	}
}

func (b *EventsChainReorg) Type() string {
	return EventsChainReorgType
}

func (b *EventsChainReorg) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsChainReorg)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsChainReorg) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsChainReorg) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
