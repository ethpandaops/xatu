package execution

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethpandaops/ethcore/pkg/execution/mimicry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/mimicry/ethereum"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestHandleBlobAnnouncements(t *testing.T) {
	var events []*xatu.DecoratedEvent

	p := &Peer{
		handlers: &handler.Peer{
			CreateNewClientMeta: func(_ context.Context) (*xatu.ClientMeta, error) {
				return &xatu.ClientMeta{Name: "test-mimicry", ClockDrift: 250}, nil
			},
			DecoratedEvent: func(_ context.Context, event *xatu.DecoratedEvent) error {
				events = append(events, event)

				return nil
			},
		},
		network:        &networks.Network{Name: networks.NetworkNameMainnet, ID: 1},
		ethereumConfig: &ethereum.Config{BlobAnnouncementsEnabled: true},
		implmentation:  "geth",
		version:        "v1.16.0-stable",
	}

	var (
		receivedAt = time.Unix(1_700_000_000, 0).UTC()
		blobA      = common.HexToHash("0xa1")
		blobB      = common.HexToHash("0xb2")
	)

	p.handleBlobAnnouncements(context.Background(), receivedAt, &mimicry.NewPooledTransactionHashes{
		Types:  []byte{types.DynamicFeeTxType, types.BlobTxType, types.LegacyTxType, types.BlobTxType},
		Sizes:  []uint32{120, 131_200, 110, 262_300},
		Hashes: []common.Hash{common.HexToHash("0x01"), blobA, common.HexToHash("0x02"), blobB},
	})

	require.Len(t, events, 2, "only blob transaction hashes are recorded")

	for i, want := range []struct {
		hash common.Hash
		size uint32
	}{{blobA, 131_200}, {blobB, 262_300}} {
		event := events[i]
		got := event.GetMempoolBlobTransactionAnnouncement()

		require.NotNil(t, got)
		assert.Equal(t, xatu.Event_MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT, event.GetEvent().GetName())
		assert.Equal(t, want.hash.Hex(), got.GetHash())
		assert.Equal(t, want.size, got.GetSize().GetValue())
		assert.Equal(t, "geth", got.GetPeerImplementation())
		assert.Equal(t, "1.16.0", got.GetPeerVersion(), "only the release version, not the operator's identity or build")
		assert.Equal(t, receivedAt.Add(250*time.Millisecond), event.GetEvent().GetDateTime().AsTime())
	}

	// eth/67 and older peers announce hashes without types, so nothing in them is known to be a blob transaction.
	events = nil

	p.handleBlobAnnouncements(context.Background(), receivedAt, &mimicry.NewPooledTransactionHashes{
		Hashes: []common.Hash{blobA},
	})

	assert.Empty(t, events)

	// Nothing is recorded unless the config enables it.
	p.ethereumConfig.BlobAnnouncementsEnabled = false

	p.handleBlobAnnouncements(context.Background(), receivedAt, &mimicry.NewPooledTransactionHashes{
		Types:  []byte{types.BlobTxType},
		Hashes: []common.Hash{blobA},
	})

	assert.Empty(t, events)
}
