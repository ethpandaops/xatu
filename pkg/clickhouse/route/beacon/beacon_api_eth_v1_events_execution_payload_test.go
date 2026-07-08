package beacon

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	colBuilderIndex        = "builder_index"
	colExecBlockHash       = "block_hash"
	colExecutionOptimistic = "execution_optimistic"
)

// TestSnapshot_beacon_api_eth_v1_events_execution_payload verifies the flat
// EIP-7732 execution_payload SSE event flattens to ClickHouse columns.
func TestSnapshot_beacon_api_eth_v1_events_execution_payload(t *testing.T) {
	if len(beaconApiEthV1EventsExecutionPayloadEventNames) == 0 {
		t.Skip("no event names registered for beacon_api_eth_v1_events_execution_payload")
	}

	blockHash := repeatHex("ee", 32)

	event := &ethv1.ExecutionPayloadEvent{
		Slot:                wrapperspb.UInt64(8_675_309),
		BuilderIndex:        wrapperspb.UInt64(42),
		BlockHash:           blockHash,
		BlockRoot:           blockRoot64A,
		ExecutionOptimistic: true,
	}

	expected := map[string]any{
		testfixture.MetaClientNameKey: testfixture.MetaClientName,
		colBlockRoot:                  blockRoot64A,
		colBuilderIndex:               uint64(42),
		colExecBlockHash:              blockHash,
		colExecutionOptimistic:        true,
	}

	testfixture.AssertSnapshot(t, newbeaconApiEthV1EventsExecutionPayloadBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     beaconApiEthV1EventsExecutionPayloadEventNames[0],
			DateTime: testfixture.TS(),
			Id:       testfixture.SnapshotID,
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_EthV1EventsExecutionPayload{
			EthV1EventsExecutionPayload: event,
		},
	}, 1, expected)
}
