package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_events_voluntary_exit(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsVoluntaryExitBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT_V2,
			DateTime: testfixture.TS(),
			Id:       "voluntary-exit-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsVoluntaryExitV2{
				EthV1EventsVoluntaryExitV2: &xatu.ClientMeta_AdditionalEthV1EventsVoluntaryExitV2Data{
					Epoch: testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsVoluntaryExitV2{
			EthV1EventsVoluntaryExitV2: &ethv1.EventVoluntaryExitV2{
				Message: &ethv1.EventVoluntaryExitMessageV2{
					Epoch:          wrapperspb.UInt64(3),
					ValidatorIndex: wrapperspb.UInt64(42),
				},
			},
		},
	}, 1, map[string]any{
		"epoch":             uint32(3),
		"validator_index":   uint32(42),
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
