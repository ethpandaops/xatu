package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	// EventsHeadV3Type carries the beacon-API head_v2 SSE topic (gloas); V3
	// because EventsHeadV2Type is the schema rev of the v1 head topic.
	EventsHeadV3Type = "BEACON_API_ETH_V1_EVENTS_HEAD_V3"
)

type EventsHeadV3 struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewEventsHeadV3(log observability.ContextualLogger, event *xatu.DecoratedEvent) *EventsHeadV3 {
	return &EventsHeadV3{
		log:   log.WithField("event", EventsHeadV3Type),
		event: event,
	}
}

func (b *EventsHeadV3) Type() string {
	return EventsHeadV3Type
}

func (b *EventsHeadV3) Validate(ctx context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_EthV1EventsHeadV3)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *EventsHeadV3) Filter(ctx context.Context) bool {
	return false
}

func (b *EventsHeadV3) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
