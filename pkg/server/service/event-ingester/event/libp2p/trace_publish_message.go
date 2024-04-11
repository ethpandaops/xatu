package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TracePublishMessageType = xatu.Event_LIBP2P_TRACE_PUBLISH_MESSAGE.String()
)

type TracePublishMessage struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTracePublishMessage(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TracePublishMessage {
	return &TracePublishMessage{
		log:   log.WithField("event", TracePublishMessageType),
		event: event,
	}
}

func (tpm *TracePublishMessage) Type() string {
	return TracePublishMessageType
}

func (tpm *TracePublishMessage) Validate(ctx context.Context) error {
	_, ok := tpm.event.Data.(*xatu.DecoratedEvent_Libp2PTracePublishMessage)
	if !ok {
		return errors.New("failed to cast event data to TracePublishMessage")
	}

	return nil
}

func (tpm *TracePublishMessage) Filter(ctx context.Context) bool {
	return false
}

func (tpm *TracePublishMessage) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
