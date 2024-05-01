package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceGossipSubBeaconAttestationType = xatu.Event_LIBP2P_TRACE_GOSSIPSUB_BEACON_ATTESTATION.String()
)

type TraceGossipSubBeaconAttestation struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubBeaconAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubBeaconAttestation {
	return &TraceGossipSubBeaconAttestation{
		log:   log.WithField("event", TraceGossipSubBeaconAttestationType),
		event: event,
	}
}

func (gsb *TraceGossipSubBeaconAttestation) Type() string {
	return TraceGossipSubBeaconAttestationType
}

func (gsb *TraceGossipSubBeaconAttestation) Validate(ctx context.Context) error {
	_, ok := gsb.event.Data.(*xatu.DecoratedEvent_Libp2PTraceGossipsubBeaconAttestation)
	if !ok {
		return errors.New("failed to cast event data to TraceGossipSubBeaconAttestation")
	}

	return nil
}

func (gsb *TraceGossipSubBeaconAttestation) Filter(ctx context.Context) bool {
	return false
}

func (gsb *TraceGossipSubBeaconAttestation) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
