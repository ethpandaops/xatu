package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_phase0_attestation_reward(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconPhase0AttestationRewardBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_PHASE0_ATTESTATION_REWARD,
			DateTime: testfixture.TS(),
			Id:       "p0ar-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconPhase0AttestationReward{
				EthV1BeaconPhase0AttestationReward: &xatu.ClientMeta_AdditionalEthV1BeaconAttestationRewardData{
					StateId: "159",
					Epoch:   testfixture.EpochAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconPhase0AttestationReward{
			EthV1BeaconPhase0AttestationReward: &xatu.AttestationRewardData{
				ValidatorIndex: wrapperspb.UInt64(42),
				Head:           wrapperspb.Int64(1000),
				Target:         wrapperspb.Int64(2000),
				Source:         wrapperspb.Int64(-500),
				InclusionDelay: wrapperspb.UInt64(3),
				Inactivity:     wrapperspb.Int64(-7),
			},
		},
	}, 1, map[string]any{
		"validator_index": uint32(42),
		"head":            int64(1000),
		"target":          int64(2000),
		"source":          int64(-500),
		"inclusion_delay": uint64(3),
		"inactivity":      int64(-7),
		"epoch":           uint32(3),
	})
}
