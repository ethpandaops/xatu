package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_canonical_beacon_blob_sidecar(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlobSidecarBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
			DateTime: testfixture.TS(),
			Id:       "cbbs-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconBlobSidecar{
				EthV1BeaconBlobSidecar: &xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconBlockBlobSidecar{
			EthV1BeaconBlockBlobSidecar: &ethv1.BlobSidecar{
				Slot:          wrapperspb.UInt64(100),
				ProposerIndex: wrapperspb.UInt64(5),
				Index:         wrapperspb.UInt64(0),
			},
		},
	}, 1, map[string]any{
		"slot":       uint32(100),
		"blob_index": uint64(0),
	})
}
