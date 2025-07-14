package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceGossipSubAggregateAndProofType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_AGGREGATE_AND_PROOF.String()
)

type TraceGossipSubAggregateAndProof struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubAggregateAndProof(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubAggregateAndProof {
	return &TraceGossipSubAggregateAndProof{
		log:   log.WithField("event", TraceGossipSubAggregateAndProofType),
		event: event,
	}
}

func (gsaap *TraceGossipSubAggregateAndProof) Type() string {
	return TraceGossipSubAggregateAndProofType
}

func (gsaap *TraceGossipSubAggregateAndProof) Validate(ctx context.Context) error {
	_, ok := gsaap.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGossipsubAggregateAndProof)
	if !ok {
		return errors.New("failed to cast event data to TraceGossipSubAggregateAndProof")
	}

	return nil
}

func (gsaap *TraceGossipSubAggregateAndProof) Filter(ctx context.Context) bool {
	return false
}

func (gsaap *TraceGossipSubAggregateAndProof) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
