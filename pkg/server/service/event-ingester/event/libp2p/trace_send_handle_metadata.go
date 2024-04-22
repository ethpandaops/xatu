package libp2p

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	TraceHandleMetadataType = xatu.Event_LIBP2P_TRACE_HANDLE_METADATA.String()
)

type TraceHandleMetadata struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewTraceHandleMetadata(log logrus.FieldLogger, event *xatu.DecoratedEvent) *TraceHandleMetadata {
	return &TraceHandleMetadata{
		log:   log.WithField("event", TraceHandleMetadataType),
		event: event,
	}
}

func (trr *TraceHandleMetadata) Type() string {
	return TraceSendRPCType
}

func (trr *TraceHandleMetadata) Validate(ctx context.Context) error {
	_, ok := trr.event.Data.(*xatu.DecoratedEvent_Libp2PTraceHandleMetadata)
	if !ok {
		return errors.New("failed to cast event data to TraceHandleMetadata")
	}

	return nil
}

func (trr *TraceHandleMetadata) Filter(ctx context.Context) bool {
	return false
}

func (trr *TraceHandleMetadata) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
