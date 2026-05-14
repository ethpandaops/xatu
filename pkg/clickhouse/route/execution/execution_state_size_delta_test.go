package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_execution_state_size_delta(t *testing.T) {
	testfixture.AssertSnapshot(t, newexecutionStateSizeDeltaBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_STATE_SIZE_DELTA,
			DateTime: testfixture.TS(),
			Id:       "essd-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_ExecutionStateSizeDelta{
			ExecutionStateSizeDelta: &xatu.ExecutionStateSizeDelta{
				Source:          "client-logs",
				BlockNumber:     wrapperspb.UInt64(1000000),
				StateRoot:       "0x0e066f3c2297a5cb300593052617d1bca5946f0caa0635fdb1b85ac7e5236f34",
				ParentStateRoot: "0xed98aa4b5b19c82fb35364f08508ae0a6dec665fa57663dca94c5d70554cde10",
				// Writes
				AccountWrites:             wrapperspb.Int64(5),
				AccountWriteBytes:         wrapperspb.Int64(305),
				AccountTrienodeWrites:     wrapperspb.Int64(23),
				AccountTrienodeWriteBytes: wrapperspb.Int64(8379),
				ContractCodeWrites:        wrapperspb.Int64(0),
				ContractCodeWriteBytes:    wrapperspb.Int64(0),
				StorageWrites:             wrapperspb.Int64(0),
				StorageWriteBytes:         wrapperspb.Int64(0),
				StorageTrienodeWrites:     wrapperspb.Int64(0),
				StorageTrienodeWriteBytes: wrapperspb.Int64(0),
				// Deletes
				AccountDeletes:             wrapperspb.Int64(5),
				AccountDeleteBytes:         wrapperspb.Int64(296),
				AccountTrienodeDeletes:     wrapperspb.Int64(23),
				AccountTrienodeDeleteBytes: wrapperspb.Int64(8370),
				ContractCodeDeletes:        wrapperspb.Int64(0),
				ContractCodeDeleteBytes:    wrapperspb.Int64(0),
				StorageDeletes:             wrapperspb.Int64(0),
				StorageDeleteBytes:         wrapperspb.Int64(0),
				StorageTrienodeDeletes:     wrapperspb.Int64(0),
				StorageTrienodeDeleteBytes: wrapperspb.Int64(0),
			},
		},
	}, 1, map[string]any{
		"block_number":                  uint64(1000000),
		"account_writes":                int64(5),
		"account_deletes":               int64(5),
		"account_write_bytes":           int64(305),
		"account_delete_bytes":          int64(296),
		"account_trienode_writes":       int64(23),
		"account_trienode_deletes":      int64(23),
		"account_trienode_write_bytes":  int64(8379),
		"account_trienode_delete_bytes": int64(8370),
		"meta_client_name":              "test-client",
	})
}
