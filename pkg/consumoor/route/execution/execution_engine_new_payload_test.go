package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
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
			ExecutionEngineNewPayload: &xatu.ExecutionEngineNewPayload{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
