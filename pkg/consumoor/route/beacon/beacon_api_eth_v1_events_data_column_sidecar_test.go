package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_data_column_sidecar(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsDataColumnSidecarBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_DATA_COLUMN_SIDECAR,
			DateTime: testfixture.TS(),
			Id:       "data-column-sidecar-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsDataColumnSidecar{
				EthV1EventsDataColumnSidecar: &xatu.ClientMeta_AdditionalEthV1EventsDataColumnSidecarData{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsDataColumnSidecar{
			EthV1EventsDataColumnSidecar: &ethv1.EventDataColumnSidecar{
				BlockRoot: "0xdcsblock",
				Slot:      wrapperspb.UInt64(100),
				Index:     wrapperspb.UInt64(7),
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"column_index":      uint64(7),
		"block_root":        "0xdcsblock",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
