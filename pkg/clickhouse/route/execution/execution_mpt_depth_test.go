package execution

import (
	"testing"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestSnapshot_execution_mpt_depth(t *testing.T) {
	testfixture.AssertSnapshot(t, newexecutionMptDepthBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_MPT_DEPTH,
			DateTime: testfixture.TS(),
			Id:       "emd-1",
		},
		Meta: testfixture.BaseMeta(),
		Data: &xatu.DecoratedEvent_ExecutionMptDepth{
			ExecutionMptDepth: &xatu.ExecutionMPTDepth{
				Source:                   "client-logs",
				BlockNumber:              wrapperspb.UInt64(1000000),
				StateRoot:                "0x0e066f3c2297a5cb300593052617d1bca5946f0caa0635fdb1b85ac7e5236f34",
				ParentStateRoot:          "0xed98aa4b5b19c82fb35364f08508ae0a6dec665fa57663dca94c5d70554cde10",
				TotalAccountWrittenNodes: wrapperspb.UInt64(23),
				TotalAccountWrittenBytes: wrapperspb.UInt64(8296),
				TotalAccountDeletedNodes: wrapperspb.UInt64(0),
				TotalAccountDeletedBytes: wrapperspb.UInt64(0),
				TotalStorageWrittenNodes: wrapperspb.UInt64(0),
				TotalStorageWrittenBytes: wrapperspb.UInt64(0),
				TotalStorageDeletedNodes: wrapperspb.UInt64(0),
				TotalStorageDeletedBytes: wrapperspb.UInt64(0),
				AccountWrittenNodes: map[uint32]uint64{
					0: 1, 1: 5, 2: 5, 3: 5, 4: 5, 5: 2,
				},
				AccountWrittenBytes: map[uint32]uint64{
					0: 532, 1: 2660, 2: 2660, 3: 1636, 4: 577, 5: 231,
				},
				AccountDeletedNodes: map[uint32]uint64{},
				AccountDeletedBytes: map[uint32]uint64{},
				StorageWrittenNodes: map[uint32]uint64{},
				StorageWrittenBytes: map[uint32]uint64{},
				StorageDeletedNodes: map[uint32]uint64{},
				StorageDeletedBytes: map[uint32]uint64{},
			},
		},
	}, 1, map[string]any{
		"block_number":                uint64(1000000),
		"total_account_written_nodes": uint64(23),
		"total_account_written_bytes": uint64(8296),
		"meta_client_name":            "test-client",
	})
}
