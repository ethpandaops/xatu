package libp2p

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceGossipSubDataColumnSidecarType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_DATA_COLUMN_SIDECAR.String()
)

type TraceGossipSubDataColumnSidecar struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubDataColumnSidecar(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubDataColumnSidecar {
	return &TraceGossipSubDataColumnSidecar{
		log:   log.WithField("event", TraceGossipSubDataColumnSidecarType),
		event: event,
	}
}

// AppendServerMeta implements event.Event.
func (t *TraceGossipSubDataColumnSidecar) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}

// Filter implements event.Event.
func (t *TraceGossipSubDataColumnSidecar) Filter(ctx context.Context) bool {
	return false
}

// Type implements event.Event.
func (t *TraceGossipSubDataColumnSidecar) Type() string {
	return TraceGossipSubDataColumnSidecarType
}

// Validate implements event.Event.
func (t *TraceGossipSubDataColumnSidecar) Validate(ctx context.Context) error {
	if _, ok := t.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGossipsubDataColumnSidecar); !ok {
		return errors.New("failed to cast event data to TraceGossipSubDataColumnSidecar")
	}

	return nil
}
