package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/consumoor/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestSnapshot_execution_state_size(t *testing.T) {
	testfixture.AssertSnapshot(t, newexecutionStateSizeBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_STATE_SIZE,
			DateTime: testfixture.TS(),
			Id:       "ess-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_ExecutionStateSize{
			ExecutionStateSize: &xatu.ExecutionStateSize{
				BlockNumber:          "1000",
				Accounts:             "500",
				AccountBytes:         "1024",
				AccountTrienodes:     "100",
				AccountTrienodeBytes: "2048",
				ContractCodes:        "50",
				ContractCodeBytes:    "4096",
				Storages:             "200",
				StorageBytes:         "8192",
				StorageTrienodes:     "150",
				StorageTrienodeBytes: "16384",
			},
		},
	}, 1, map[string]any{
		"meta_client_name": "test-client",
	})
}
