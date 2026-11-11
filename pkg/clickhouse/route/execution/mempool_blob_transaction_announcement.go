package execution

import (
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var mempoolBlobTransactionAnnouncementEventNames = []xatu.Event_Name{
	xatu.Event_MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT,
}

func init() {
	r, err := route.NewStaticRoute(
		mempoolBlobTransactionAnnouncementTableName,
		mempoolBlobTransactionAnnouncementEventNames,
		func() route.ColumnarBatch { return newmempoolBlobTransactionAnnouncementBatch() },
	)
	if err != nil {
		route.RecordError(err)

		return
	}

	if err := route.Register(r); err != nil {
		route.RecordError(err)
	}
}

func (b *mempoolBlobTransactionAnnouncementBatch) FlattenTo(event *xatu.DecoratedEvent) error {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	payload := event.GetMempoolBlobTransactionAnnouncement()
	if payload == nil || payload.GetHash() == "" {
		return fmt.Errorf("mempool_blob_transaction_announcement without hash: %w", route.ErrInvalidEvent)
	}

	b.UpdatedDateTime.Append(time.Now())
	b.EventDateTime.Append(event.GetEvent().GetDateTime().AsTime())
	b.Hash.Append([]byte(payload.GetHash()))
	b.Size.Append(payload.GetSize().GetValue())
	b.PeerImplementation.Append(payload.GetPeerImplementation())
	b.PeerVersion.Append(payload.GetPeerVersion())
	b.appendMetadata(event)
	b.rows++

	return nil
}
