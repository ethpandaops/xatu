package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
			ExecutionEngineGetBlobs: &xatu.ExecutionEngineGetBlobs{
				DurationMs:     wrapperspb.UInt64(10),
				RequestedCount: wrapperspb.UInt32(3),
				ReturnedCount:  wrapperspb.UInt32(2),
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
