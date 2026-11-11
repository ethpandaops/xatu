package execution

import (
	"context"
	"regexp"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// semverPattern finds the release version in a devp2p client name's version part.
var semverPattern = regexp.MustCompile(`\d+\.\d+\.\d+`)

// peerSemver keeps only the release version of a peer's devp2p name, such as "1.16.0" from
// "myrig/v1.16.0-stable-abc123/linux-amd64/go1.24", so operator identities and build strings are not recorded.
func peerSemver(version string) string {
	return semverPattern.FindString(version)
}

// handleBlobAnnouncements emits a MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT event for every blob transaction hash in a
// peer's announcement when ethereum.blobAnnouncementsEnabled is set. The event names the peer only by its client
// implementation and release version.
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
			PeerVersion:        peerSemver(p.version),
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
