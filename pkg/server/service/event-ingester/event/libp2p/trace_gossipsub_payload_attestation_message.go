package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	TraceGossipSubPayloadAttestationMessageType = "LIBP2P_TRACE_GOSSIPSUB_PAYLOAD_ATTESTATION_MESSAGE"
)

type TraceGossipSubPayloadAttestationMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceGossipSubPayloadAttestationMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceGossipSubPayloadAttestationMessage {
	return &TraceGossipSubPayloadAttestationMessage{
		log:   log.WithField("event", TraceGossipSubPayloadAttestationMessageType),
		event: event,
	}
}

func (e *TraceGossipSubPayloadAttestationMessage) Type() string {
	return TraceGossipSubPayloadAttestationMessageType
}

func (e *TraceGossipSubPayloadAttestationMessage) Validate(_ context.Context) error {
	_, ok := e.event.GetData().(*xatu.DecoratedEvent_Libp2PTraceGossipsubPayloadAttestationMessage)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (e *TraceGossipSubPayloadAttestationMessage) Filter(_ context.Context) bool {
	return false
}

func (e *TraceGossipSubPayloadAttestationMessage) AppendServerMeta(_ context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
