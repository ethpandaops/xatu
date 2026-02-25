package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_execution_engine_new_payload(t *testing.T) {
	testfixture.AssertSnapshot(t, newexecutionEngineNewPayloadBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_ENGINE_NEW_PAYLOAD,
			DateTime: testfixture.TS(),
			Id:       "eenp-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_ExecutionEngineNewPayload{
			ExecutionEngineNewPayload: &xatu.ExecutionEngineNewPayload{
				DurationMs:  wrapperspb.UInt64(10),
				BlockNumber: wrapperspb.UInt64(1000),
				GasUsed:     wrapperspb.UInt64(21000),
				GasLimit:    wrapperspb.UInt64(30000000),
				TxCount:     wrapperspb.UInt32(5),
				BlobCount:   wrapperspb.UInt32(2),
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
