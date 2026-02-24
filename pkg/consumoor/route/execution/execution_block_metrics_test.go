package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_execution_block_metrics(t *testing.T) {
	testfixture.AssertSnapshot(t, newexecutionBlockMetricsBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_BLOCK_METRICS,
			DateTime: testfixture.TS(),
			Id:       "ebm-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_ExecutionBlockMetrics{
			ExecutionBlockMetrics: &xatu.ExecutionBlockMetrics{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
