package execution

import (
	"testing"

	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route/testfixture"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	colBlockNumber    = "block_number"
	colBlockHash      = "block_hash"
	colGasUsed        = "gas_used"
	colMetaNetwork    = "meta_network_name"
	testBlockHashHex  = "0xfc429da12e414c0e8348fd9bb760b1a118b4dcf73419f021d4ccc2c3b793e05c"
	testAuthorHex     = "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97"
	testParentHashHex = "0x1b1a5b2b63d5db1d26d30a2a2cff0e8e2b2e4c9d0f5b4f3c2a1b0c9d8e7f6a5b"
	testStateRootHex  = "0x2c2b6c3c74e6ec2e37e41b3b3d00f9f93c3f5daea06c5a4d3b2c1dae9f807b6c"
)

func TestSnapshot_canonical_execution_block(t *testing.T) {
	testfixture.AssertSnapshot(t, newCanonicalExecutionBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_BLOCK,
			DateTime: testfixture.TS(),
			Id:       "ceb-1",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: &xatu.ExecutionCanonicalBlock{
				Blocks: []*xatu.ExecutionBlock{
					{
						BlockNumber:     22000000,
						BlockHash:       testBlockHashHex,
						BlockDateTime:   timestamppb.New(testfixture.TS().AsTime()),
						Author:          wrapperspb.String(testAuthorHex),
						GasUsed:         wrapperspb.UInt64(19291457),
						ExtraData:       wrapperspb.String("0x546974616e"),
						ExtraDataString: wrapperspb.String("Titan"),
						BaseFeePerGas:   wrapperspb.UInt64(611253386),
						ParentHash:      wrapperspb.String(testParentHashHex),
						StateRoot:       wrapperspb.String(testStateRootHex),
						BlobGasUsed:     wrapperspb.UInt64(786432),
						ExcessBlobGas:   wrapperspb.UInt64(0),
					},
				},
			},
		},
	}, 1, map[string]any{
		colBlockNumber:      uint64(22000000),
		colBlockHash:        testBlockHashHex,
		"author":            testAuthorHex,
		colGasUsed:          uint64(19291457),
		"extra_data_string": "Titan",
		"base_fee_per_gas":  uint64(611253386),
		"parent_hash":       testParentHashHex,
		"state_root":        testStateRootHex,
		"blob_gas_used":     uint64(786432),
		"excess_blob_gas":   uint64(0),
		colMetaNetwork:      "mainnet",
	})
}

func TestSnapshot_canonical_execution_block_without_header_extras(t *testing.T) {
	testfixture.AssertSnapshot(t, newCanonicalExecutionBlockBatch(), &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_EXECUTION_CANONICAL_BLOCK,
			DateTime: testfixture.TS(),
			Id:       "ceb-2",
		},
		Meta: testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: &xatu.ExecutionCanonicalBlock{
				Blocks: []*xatu.ExecutionBlock{{BlockNumber: 1, BlockHash: testBlockHashHex}},
			},
		},
	}, 1, map[string]any{
		"parent_hash":     nil,
		"state_root":      nil,
		"blob_gas_used":   nil,
		"excess_blob_gas": nil,
	})
}

// TestFlattenTo_canonical_execution_block_chunk verifies a multi-block chunk
// flattens to one row per block.
func TestFlattenTo_canonical_execution_block_chunk(t *testing.T) {
	batch := newCanonicalExecutionBlockBatch()

	blocks := make([]*xatu.ExecutionBlock, 0, 5)
	for i := uint64(0); i < 5; i++ {
		blocks = append(blocks, &xatu.ExecutionBlock{
			BlockNumber:   22000000 + i,
			BlockHash:     testBlockHashHex,
			BlockDateTime: timestamppb.New(testfixture.TS().AsTime()),
		})
	}

	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{Name: xatu.Event_EXECUTION_CANONICAL_BLOCK, DateTime: testfixture.TS(), Id: "ceb-chunk"},
		Meta:  testfixture.MetaWithAdditional(&xatu.ClientMeta{}),
		Data: &xatu.DecoratedEvent_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: &xatu.ExecutionCanonicalBlock{Blocks: blocks},
		},
	}

	if err := batch.FlattenTo(event); err != nil {
		t.Fatalf("FlattenTo: %v", err)
	}

	if batch.Rows() != 5 {
		t.Fatalf("expected 5 rows, got %d", batch.Rows())
	}
}
