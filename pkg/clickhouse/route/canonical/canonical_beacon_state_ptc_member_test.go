package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_ptc_member(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStatePtcMemberBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER,
			DateTime: testfixture.TS(),
			Id:       "ptc-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStatePtcMember{
				EthV1BeaconStatePtcMember: &xatu.ClientMeta_AdditionalEthV1BeaconStatePtcMemberData{
					Slot:    testfixture.SlotEpochAdditional(),
					Epoch:   testfixture.EpochAdditional(),
					StateId: "100",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStatePtcMember{
			EthV1BeaconStatePtcMember: &xatu.PtcMemberData{
				Position:       wrapperspb.UInt64(511),
				ValidatorIndex: wrapperspb.UInt64(85272),
			},
		},
	}, 1, map[string]any{
		"slot":            uint32(100),
		"epoch":           uint32(3),
		"state_id":        "100",
		"position":        uint32(511),
		"validator_index": uint32(85272),
	})
}
