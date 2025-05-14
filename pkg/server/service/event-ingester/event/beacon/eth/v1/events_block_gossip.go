package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsBlockGossipType = "BEACON_API_ETH_V1_EVENTS_BLOCK_GOSSIP"
)

type EventsBlockGossip struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsBlockGossip(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsBlockGossip {
	return &EventsBlockGossip{
		log:   log.WithField("event", EventsBlockGossipType),
		event: event,
	}
}

func (b *EventsBlockGossip) Type() string {
	return EventsBlockGossipType
}

func (b *EventsBlockGossip) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsBlockGossip)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsBlockGossip) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsBlockGossip) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
