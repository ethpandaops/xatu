package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	TraceGossipSubExecutionPayloadBidType = "LIBP2P_TRACE_GOSSIPSUB_EXECUTION_PAYLOAD_BID"
)

type TraceGossipSubExecutionPayloadBid struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubExecutionPayloadBid(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubExecutionPayloadBid {
	return &TraceGossipSubExecutionPayloadBid{
		log:   log.WithField("event", TraceGossipSubExecutionPayloadBidType),
		event: event,
	}
}

func (e *TraceGossipSubExecutionPayloadBid) Type() string {
	return TraceGossipSubExecutionPayloadBidType
}

func (e *TraceGossipSubExecutionPayloadBid) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceGossipsubExecutionPayloadBid)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *TraceGossipSubExecutionPayloadBid) Filter(_ context.Context) bool {
	return false
}

func (e *TraceGossipSubExecutionPayloadBid) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
