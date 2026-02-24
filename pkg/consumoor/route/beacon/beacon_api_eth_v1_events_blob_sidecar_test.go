package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_blob_sidecar(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsBlobSidecarBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOB_SIDECAR,
			DateTime: testfixture.TS(),
			Id:       "blob-sidecar-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsBlobSidecar{
				EthV1EventsBlobSidecar: &xatu.ClientMeta_AdditionalEthV1EventsBlobSidecarData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsBlobSidecar{
			EthV1EventsBlobSidecar: &ethv1.EventBlobSidecar{
				BlockRoot:     "0xbr",
				Slot:          wrapperspb.UInt64(100),
				Index:         wrapperspb.UInt64(2),
				KzgCommitment: "0xkzg",
				VersionedHash: "0xvh",
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"blob_index":        uint64(2),
		"block_root":        "0xbr",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
