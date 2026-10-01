package canonical

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_beacon_state_execution_payload_availability(t *testing.T) {
	testfixture.AssertSnapshot(t, newcanonicalBeaconStateExecutionPayloadAvailabilityBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY,
			DateTime: testfixture.TS(),
			Id:       "epa-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1BeaconStateExecutionPayloadAvailability{
				EthV1BeaconStateExecutionPayloadAvailability: &xatu.ClientMeta_AdditionalEthV1BeaconStateExecutionPayloadAvailabilityData{
					Slot:    testfixture.SlotEpochAdditional(),
					Epoch:   testfixture.EpochAdditional(),
					StateId: "160",
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1BeaconStateExecutionPayloadAvailability{
			EthV1BeaconStateExecutionPayloadAvailability: &xatu.ExecutionPayloadAvailabilityData{
				Available: wrapperspb.Bool(true),
			},
		},
	}, 1, map[string]any{
		"slot":      uint32(100),
		"epoch":     uint32(3),
		"state_id":  "160",
		"available": true,
	})
}
