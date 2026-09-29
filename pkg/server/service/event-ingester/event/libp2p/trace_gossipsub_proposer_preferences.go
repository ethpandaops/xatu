package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	TraceGossipSubProposerPreferencesType = "LIBP2P_TRACE_GOSSIPSUB_PROPOSER_PREFERENCES"
)

type TraceGossipSubProposerPreferences struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubProposerPreferences(log observability.ContextualLogger, event *xatu.DecoratedEvent) *TraceGossipSubProposerPreferences {
	return &TraceGossipSubProposerPreferences{
		log:   log.WithField("event", TraceGossipSubProposerPreferencesType),
		event: event,
	}
}

func (e *TraceGossipSubProposerPreferences) Type() string {
	return TraceGossipSubProposerPreferencesType
}

func (e *TraceGossipSubProposerPreferences) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceGossipsubProposerPreferences)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *TraceGossipSubProposerPreferences) Filter(_ context.Context) bool {
	return false
}

func (e *TraceGossipSubProposerPreferences) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
