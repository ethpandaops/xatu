package beacon

import (
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// repeatHex builds a 0x-prefixed hex string from a repeated byte pair.
func repeatHex(pair string, count int) string {
	return "0x" + strings.Repeat(pair, count)
}

func TestSnapshot_beacon_api_eth_v1_events_execution_payload_available(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadAvailableEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload_available")
	}

	const (
		colEpoch                    = "epoch"
		colPropagationSlotStartDiff = "propagation_slot_start_diff"
	)

	blockRoot := repeatHex("1a", 32)

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadAvailableBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadAvailableEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV1EventsExecutionPayloadAvailable{
				EthV1EventsExecutionPayloadAvailable: &xatu.ClientMeta_AdditionalEthV1EventsExecutionPayloadAvailableData{
					Slot:        testfixture.SlotEpochAdditional(),
					Epoch:       testfixture.EpochAdditional(),
					Propagation: testfixture.PropagationAdditional(),
				},
			},
		}),
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayloadAvailable{
			EthV1EventsExecutionPayloadAvailable: &ethv1.ExecutionPayloadAvailable{
				BlockRoot: blockRoot,
				Slot:      wrapperspb.UInt64(48752),
			},
		},
	}, 1, map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colBlockRoot:                  blockRoot,
		colSlot:                       uint32(100),
		colEpoch:                      uint32(3),
		colPropagationSlotStartDiff:   uint32(500),
	})
}
