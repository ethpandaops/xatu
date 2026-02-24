package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_execution_engine_get_blobs(t *testing.T) {
	testfixture.AssertSnapshot(t, newexecutionEngineGetBlobsBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_ENGINE_GET_BLOBS,
			DateTime: testfixture.TS(),
			Id:       "eegb-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_ExecutionEngineGetBlobs{
			ExecutionEngineGetBlobs: &xatu.ExecutionEngineGetBlobs{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
