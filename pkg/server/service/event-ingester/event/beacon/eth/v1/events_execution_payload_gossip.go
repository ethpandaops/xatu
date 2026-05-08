package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsExecutionPayloadGossipType = "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_GOSSIP"
)

type EventsExecutionPayloadGossip struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsExecutionPayloadGossip(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsExecutionPayloadGossip {
	return &EventsExecutionPayloadGossip{
		log:   log.WithField("event", EventsExecutionPayloadGossipType),
		event: event,
	}
}

func (e *EventsExecutionPayloadGossip) Type() string {
	return EventsExecutionPayloadGossipType
}

func (e *EventsExecutionPayloadGossip) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsExecutionPayloadGossip)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *EventsExecutionPayloadGossip) Filter(_ context.Context) bool {
	return false
}

func (e *EventsExecutionPayloadGossip) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
