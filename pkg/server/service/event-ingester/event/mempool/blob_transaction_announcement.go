package mempool

import (
	"context"
	"errors"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BlobTransactionAnnouncementType = "MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT"
)

type BlobTransactionAnnouncement struct {
	log   observability.ContextualLogger
	event *xatu.DecoratedEvent
}

func NewBlobTransactionAnnouncement(log observability.ContextualLogger, event *xatu.DecoratedEvent) *BlobTransactionAnnouncement {
	return &BlobTransactionAnnouncement{
		log:   log.WithField("event", BlobTransactionAnnouncementType),
		event: event,
	}
}

func (b *BlobTransactionAnnouncement) Type() string {
	return BlobTransactionAnnouncementType
}

func (b *BlobTransactionAnnouncement) Validate(ctx context.Context) error {
	if b.event.GetMempoolBlobTransactionAnnouncement() == nil {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BlobTransactionAnnouncement) Filter(ctx context.Context) bool {
	return false
}

func (b *BlobTransactionAnnouncement) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	return meta
}
