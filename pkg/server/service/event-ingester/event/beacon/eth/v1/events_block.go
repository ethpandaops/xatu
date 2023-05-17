package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsBlockType = "BEACON_API_ETH_V1_EVENTS_BLOCK"
)

type EventsBlock struct {
	log       logrus.FieldLogger
	event     *xatu.DecoratedEvent
	networkID uint64
}

func NewEventsBlock(log logrus.FieldLogger, event *xatu.DecoratedEvent, networkID uint64) *EventsBlock {
	return &EventsBlock{
		log:       log.WithField("event", EventsBlockType),
		event:     event,
		networkID: networkID,
	}
}

func (b *EventsBlock) Type() string {
	return EventsBlockType
}

func (b *EventsBlock) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsBlock)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsBlock) Filter(ctx context.Context) bool {
	networkID := b.event.GetMeta().GetClient().GetEthereum().GetNetwork().GetId()

	return networkID != b.networkID
}
