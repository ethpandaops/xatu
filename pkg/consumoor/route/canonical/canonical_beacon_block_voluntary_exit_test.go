package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_block_voluntary_exit(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconBlockVoluntaryExitBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT,
			DateTime: testfixture.TS(),
			Id:       "cbve-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV2BeaconBlockVoluntaryExit{
				EthV2BeaconBlockVoluntaryExit: &xatu.ClientMeta_AdditionalEthV2BeaconBlockVoluntaryExitData{
					Block: &xatu.BlockIdentifier{
						Epoch: testfixture.EpochAdditional(),
					},
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &ethv1.SignedVoluntaryExitV2{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
