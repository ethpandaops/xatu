package node

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	noderecord "github.com/ethpandaops/xatu/pkg/proto/noderecord"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_node_record_execution(t *testing.T) {
	testfixture.AssertSnapshot(t, newnodeRecordExecutionBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_NODE_RECORD_EXECUTION,
			DateTime: testfixture.TS(),
			Id:       "nre-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_NodeRecordExecution{
			NodeRecordExecution: &noderecord.Execution{},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
