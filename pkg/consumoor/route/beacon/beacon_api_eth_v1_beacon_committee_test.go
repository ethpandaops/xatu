package beacon

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_beacon_api_eth_v1_beacon_committee(t *testing.T) {
	testfixture.AssertSnapshot(t, newbeaconApiEthV1BeaconCommitteeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_COMMITTEE,
			DateTime: testfixture.TS(),
			Id:       "beacon-committee-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconCommittee{
				EthV1BeaconCommittee: &xatu.ClientMeta_AdditionalEthV1BeaconCommitteeData{
					StateId: "head",
					Slot:    testfixture.SlotEpochAdditional(),
					Epoch:   testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconCommittee{
			EthV1BeaconCommittee: &ethv1.Committee{
				Index: wrapperspb.UInt64(5),
				Slot:  wrapperspb.UInt64(100),
			},
		},
	}, 1, map[string]any{
		"slot":              uint32(100),
		"committee_index":   "5",
		"meta_client_name":  "test-client",
		"meta_network_name": "mainnet",
	})
}
