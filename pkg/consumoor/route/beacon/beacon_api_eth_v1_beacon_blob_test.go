package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_beacon_blob(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1BeaconBlobBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB,
			DateTime: testfixture.TS(),
			Id:       "beacon-blob-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconBlob{
				EthV1BeaconBlob: &xatu.ClientMeta_AdditionalEthV1BeaconBlobData{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconBlob{
			EthV1BeaconBlob: &ethv1.Blob{
				Slot:  wrapperspb.UInt64(100),
				Index: wrapperspb.UInt64(1),
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"blob_index":        uint64(1),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
