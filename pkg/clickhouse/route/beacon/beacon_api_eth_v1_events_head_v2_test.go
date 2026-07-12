package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	colBlock           = "block"
	colEpochTransition = "epoch_transition"
	colMetaClientName  = "meta_client_name"
	colMetaNetworkName = "meta_network_name"
	testClientName     = "test-client"
	testNetworkName    = "mainnet"
	testBlockRoot      = "0xblock1"
)

func TestSnapshot_beacon_api_eth_v1_events_head_v2(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsHeadV2Batch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V3,
			DateTime: testfixture.TS(),
			Id:       "head-v2-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsHeadV3{
				EthV1EventsHeadV3: &xatu.ClientMeta_AdditionalEthV1EventsHeadV3Data{
					Slot:  testfixture.SlotEpochAdditional(),
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsHeadV3{
			EthV1EventsHeadV3: &ethv1.EventHeadV3{
				Slot:                      wrapperspb.UInt64(100),
				Block:                     testBlockRoot,
				State:                     "0xstate1",
				PayloadStatus:             "full",
				EpochTransition:           true,
				CurrentEpochDependentRoot: "0xcurrent1",
				NextEpochDependentRoot:    "0xnext1",
				ExecutionOptimistic:       true,
			},
		},
	}, 1, map[string]any{
		colSlot:                        uint32(100),
		colBlock:                       testBlockRoot,
		"payload_status":               "full",
		colEpochTransition:             true,
		colExecutionOptimistic:         true,
		"current_epoch_dependent_root": "0xcurrent1",
		"next_epoch_dependent_root":    "0xnext1",
		colMetaClientName:              testClientName,
		colMetaNetworkName:             testNetworkName,
	})
}
