package mevrelay

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

var (
	BidTraceBuilderBlockSubmissionType = xatu.Event_MEV_RELAY_BID_TRACE_BUILDER_BLOCK_SUBMISSION.String()
)

type BidTraceBuilderBlockSubmission struct {
	log   logrus.FieldLogger
	event *xatu.DecoratedEvent
}

func NewBidTraceBuilderBlockSubmission(log logrus.FieldLogger, event *xatu.DecoratedEvent) *BidTraceBuilderBlockSubmission {
	return &BidTraceBuilderBlockSubmission{
		log:   log.WithField("event", BidTraceBuilderBlockSubmissionType),
		event: event,
	}
}

func (b *BidTraceBuilderBlockSubmission) Type() string {
	return BidTraceBuilderBlockSubmissionType
}

func (b *BidTraceBuilderBlockSubmission) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_MevRelayBidTraceBuilderBlockSubmission)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BidTraceBuilderBlockSubmission) Filter(ctx context.Context) bool {
	return false
}

func (b *BidTraceBuilderBlockSubmission) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
