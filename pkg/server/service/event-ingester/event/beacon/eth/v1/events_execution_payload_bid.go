package v1

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	EventsExecutionPayloadBidType = "BEACON_API_ETH_V1_EVENTS_EXECUTION_PAYLOAD_BID"
)

type EventsExecutionPayloadBid struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewEventsExecutionPayloadBid(log logrus.FieldLogger, event *xatu.DecoratedEvent) *EventsExecutionPayloadBid {
	return &EventsExecutionPayloadBid{
		log:   log.WithField("event", EventsExecutionPayloadBidType),
		event: event,
	}
}

func (e *EventsExecutionPayloadBid) Type() string {
	return EventsExecutionPayloadBidType
}

func (e *EventsExecutionPayloadBid) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_EthV1EventsExecutionPayloadBid)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *EventsExecutionPayloadBid) Filter(_ context.Context) bool {
	return false
}

func (e *EventsExecutionPayloadBid) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
