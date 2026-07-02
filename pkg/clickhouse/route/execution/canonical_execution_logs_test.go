package execution

import (
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_canonical_execution_logs(t *testing.T) {
	testfixture.AssertSnapshot(t, newCanonicalExecutionLogsBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_EXECUTION_CANONICAL_LOGS, DateTime: testfixture.TS(), Id: "cel-1"},
		Meta:  testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_ExecutionCanonicalLogs{
			ExecutionCanonicalLogs: &xatu.ExecutionCanonicalLogs{
				Logs: []*xatu.ExecutionLog{
					{
						BlockNumber:      22000000,
						TransactionIndex: 5,
						TransactionHash:  testBlockHashHex,
						InternalIndex:    1,
						LogIndex:         193,
						Address:          "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
						Topic0:           "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef",
						Topic1:           wrapperspb.String("0x0000000000000000000000001ef032a3c471a99cc31578c8007f256d95e89896"),
						// topic2/topic3/data nil → NULL
					},
				},
			},
		},
	}, 1, map[string]any{
		"block_number":   uint64(22000000),
		"internal_index": uint32(1),
		"log_index":      uint32(193),
		"address":        "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
		"topic1":         "0x0000000000000000000000001ef032a3c471a99cc31578c8007f256d95e89896",
		"topic2":         nil,
		"data":           nil,
	})
}

func TestSnapshot_canonical_execution_traces(t *testing.T) {
	testfixture.AssertSnapshot(t, newCanonicalExecutionTracesBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_EXECUTION_CANONICAL_TRACES, DateTime: testfixture.TS(), Id: "cet-1"},
		Meta:  testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_ExecutionCanonicalTraces{
			ExecutionCanonicalTraces: &xatu.ExecutionCanonicalTraces{
				Traces: []*xatu.ExecutionTrace{
					{
						BlockNumber:      22000000,
						TransactionIndex: 77,
						TransactionHash:  testBlockHashHex,
						InternalIndex:    1,
						ActionFrom:       "0xb1b2d032aa2f52347fbcfd08e5c3cc55216e8404",
						ActionTo:         wrapperspb.String("0x1ef032a3c471a99cc31578c8007f256d95e89896"),
						ActionValue:      "1000000000000000000",
						ActionGas:        50000,
						ActionType:       "call",
						ResultGasUsed:    21000,
					},
				},
			},
		},
	}, 1, map[string]any{
		"block_number":   uint64(22000000),
		"internal_index": uint32(1),
		"action_from":    "0xb1b2d032aa2f52347fbcfd08e5c3cc55216e8404",
		"action_to":      "0x1ef032a3c471a99cc31578c8007f256d95e89896",
		"action_value":   "1000000000000000000",
		"action_gas":     uint64(50000),
		"action_type":    "call",
	})
}
