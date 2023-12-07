package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsHeadV2Type = "BEACON_API_ETH_V1_EVENTS_HEAD_V2"
)

type EventsHeadV2 struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsHeadV2(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsHeadV2 {
	return &EventsHeadV2{
		log:   log.WithField("event", EventsHeadV2Type),
		event: event,
	}
}

func (b *EventsHeadV2) Type() string {
	return EventsHeadV2Type
}

func (b *EventsHeadV2) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsHeadV2)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsHeadV2) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsHeadV2) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
