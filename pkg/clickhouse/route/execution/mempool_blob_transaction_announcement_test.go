package execution

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_mempool_blob_transaction_announcement(t *testing.T) {
	testfixture.AssertSnapshot(t, newmempoolBlobTransactionAnnouncementBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_MEMPOOL_BLOB_TRANSACTION_ANNOUNCEMENT,
			DateTime: testfixture.TS(),
			Id:       "mbta-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_MempoolBlobTransactionAnnouncement{
			MempoolBlobTransactionAnnouncement: &xatu.MempoolBlobTransactionAnnouncement{
				Hash:               "0x88a4c68fdb528cdbb40bb10ee0c640c8026da927564a6640098fd30613e70433",
				Size:               wrapperspb.UInt32(131_200),
				PeerImplementation: "geth",
				PeerVersion:        "v1.16.0-stable",
			},
		},
	}, 1, map[string]any{
		"hash":                "0x88a4c68fdb528cdbb40bb10ee0c640c8026da927564a6640098fd30613e70433",
		"size":                uint32(131_200),
		"peer_implementation": "geth",
		"peer_version":        "v1.16.0-stable",
		"meta_client_name":    "test-client",
	})
}
