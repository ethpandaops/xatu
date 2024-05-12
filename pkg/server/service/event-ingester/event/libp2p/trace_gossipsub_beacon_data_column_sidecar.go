package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceGossipSubBeaconDataColumnSidecarType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_DATA_COLUMN_SIDECAR.String()
)

type TraceGossipSubBeaconDataColumnSidecar struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubBeaconDataColumnSidecar(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubBeaconDataColumnSidecar {
	return &TraceGossipSubBeaconDataColumnSidecar{
		log:   log.WithField("event", TraceGossipSubBeaconDataColumnSidecarType),
		event: event,
	}
}

func (gsb *TraceGossipSubBeaconDataColumnSidecar) Type() string {
	return TraceGossipSubBeaconDataColumnSidecarType
}

func (gsb *TraceGossipSubBeaconDataColumnSidecar) Validate(ctx context.Context) error {
	_, ok := gsb.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconAttestation)
	if !ok {
		return errors.New("failed to cast event data to TraceGossipSubBeaconDataColumnSidecar")
	}

	return nil
}

func (gsb *TraceGossipSubBeaconDataColumnSidecar) Filter(ctx context.Context) bool {
	return false
}

func (gsb *TraceGossipSubBeaconDataColumnSidecar) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
