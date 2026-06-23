package canonical

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_randao(t *testing.T) {
	randao := "0x1111111111111111111111111111111111111111111111111111111111111111"

	testfixture.AssertSnapshot(t, newcanonicalBeaconStateRandaoBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_RANDAO,
			DateTime: testfixture.TS(),
			Id:       "randao-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStateRandao{
				EthV1BeaconStateRandao: &xatu.ClientMeta_AdditionalEthV1BeaconStateRandaoData{
					Epoch:   testfixture.EpochAdditional(),
					StateId: "96",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStateRandao{
			EthV1BeaconStateRandao: &xatu.RandaoData{
				Randao: randao,
			},
		},
	}, 1, map[string]any{
		"epoch":    uint32(3),
		"state_id": "96",
		"randao":   randao,
	})
}
