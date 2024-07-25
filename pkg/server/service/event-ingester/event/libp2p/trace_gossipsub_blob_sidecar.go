package libp2p

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	TraceGossipSubBlobSidecarType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BLOB_SIDECAR.String()
)

type TraceGossipSubBlobSidecar struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubBlobSidecar(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubBlobSidecar {
	return &TraceGossipSubBlobSidecar{
		log:   log.WithField("event", TraceGossipSubBlobSidecarType),
		event: event,
	}
}

// AppendServerMeta implements event.Event.
func (t *TraceGossipSubBlobSidecar) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}

// Filter implements event.Event.
func (t *TraceGossipSubBlobSidecar) Filter(ctx context.Context) bool {
	return false
}

// Type implements event.Event.
func (t *TraceGossipSubBlobSidecar) Type() string {
	return TraceGossipSubBlobSidecarType
}

// Validate implements event.Event.
func (t *TraceGossipSubBlobSidecar) Validate(ctx context.Context) error {
	if _, ok := t.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGossipsubBlobSidecar); !ok {
		return errors.New("failed to cast event data to TraceGossipSubBlobSidecar")
	}

	return nil
}
