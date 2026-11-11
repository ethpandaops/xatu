package execution

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// handleBlobAnnouncements emits a MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT event for every blob transaction hash in a
// peer's announcement when ethereum.blobAnnouncementsEnabled is set. The event names the peer only by its client
// implementation and version.
func (p *Peer) handleBlobAnnouncements(ctx context.Context, receivedAt time.Time, hashes *mimicry.NewPooledTransactionHashes) {
	if p.ethereumConfig == nil || !p.ethereumConfig.BlobAnnouncementsEnabled {
		return
	}

	for i, hash := range hashes.Hashes {
		if i >= len(hashes.Types) || hashes.Types[i] != types.BlobTxType {
			continue
		}

		meta, err := p.createNewClientMeta(ctx)
		if err != nil {
			p.log.WithError(err).WithContext(ctx).Error("Failed to create client meta")

			return
		}

		announcement := &xatu.MempoolBlobTransactionAnnouncement{
			Hash:               hash.String(),
			PeerImplementation: p.implmentation,
			PeerVersion:        p.version,
		}

		if i < len(hashes.Sizes) {
			announcement.Size = wrapperspb.UInt32(hashes.Sizes[i])
		}

		event := &xatu.DecoratedEvent{
			Event: &xatu.Event{
				Name:     xatu.Event_MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT,
				DateTime: timestamppb.New(receivedAt.Add(time.Duration(meta.GetClockDrift()) * time.Millisecond)), //nolint:gosec // drift is milliseconds
				Id:       uuid.New().String(),
			},
			Meta: &xatu.Meta{Client: meta},
			Data: &xatu.DecoratedEvent_MempoolBlobTransactionAnnouncement{
				MempoolBlobTransactionAnnouncement: announcement,
			},
		}

		if err := p.handlers.DecoratedEvent(ctx, event); err != nil {
			p.log.WithError(err).WithContext(ctx).Error("Failed to handle blob transaction announcement")
		}
	}
}
