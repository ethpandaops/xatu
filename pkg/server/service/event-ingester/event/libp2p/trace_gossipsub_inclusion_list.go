package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	TraceGossipSubInclusionListType = "LIBP2P_TRACE_GOSSIPSUB_INCLUSION_LIST"
)

type TraceGossipSubInclusionList struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubInclusionList(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubInclusionList {
	return &TraceGossipSubInclusionList{
		log:   log.WithField("event", TraceGossipSubInclusionListType),
		event: event,
	}
}

func (e *TraceGossipSubInclusionList) Type() string {
	return TraceGossipSubInclusionListType
}

func (e *TraceGossipSubInclusionList) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceGossipsubInclusionList)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *TraceGossipSubInclusionList) Filter(_ context.Context) bool {
	return false
}

func (e *TraceGossipSubInclusionList) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
