package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	testBlockHash  = "0xfc429da12e414c0e8348fd9bb760b1a118b4dcf73419f021d4ccc2c3b793e05c"
	testParentHash = "0x1b1a5b2b63d5db1d26d30a2a2cff0e8e2b2e4c9d0f5b4f3c2a1b0c9d8e7f6a5b"
	testStateRoot  = "0x2c2b6c3c74e6ec2e37e41b3b3d00f9f93c3f5daea06c5a4d3b2c1dae9f807b6c"
)

func TestRowToProto(t *testing.T) {
	row := &blockRow{
		BlockNumber:   22000000,
		BlockHash:     testBlockHash,
		Author:        "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97",
		GasUsed:       19291457,
		GasLimit:      35964811,
		ExtraData:     "0x546974616e", // "Titan"
		Timestamp:     1741410875,
		BaseFeePerGas: 611253386,
		ParentHash:    testParentHash,
		StateRoot:     testStateRoot,
		BlobGasUsed:   uint64Ptr(786432),
		ExcessBlobGas: uint64Ptr(0),
	}

	out := blockRowToProto(row)

	assert.Equal(t, uint64(22000000), out.GetBlockNumber())
	assert.Equal(t, testBlockHash, out.GetBlockHash())
	assert.Len(t, out.GetBlockHash(), 66, "block hash must be 0x + 64 hex chars for FixedString(66)")
	assert.Equal(t, "0x4838b106fce9647bdf1e7877bf73ce8b0bad5f97", out.GetAuthor().GetValue())
	assert.Equal(t, uint64(19291457), out.GetGasUsed().GetValue())
	assert.Equal(t, uint64(35964811), out.GetGasLimit().GetValue())
	assert.Equal(t, uint64(611253386), out.GetBaseFeePerGas().GetValue())
	assert.Equal(t, "0x546974616e", out.GetExtraData().GetValue())
	assert.Equal(t, "Titan", out.GetExtraDataString().GetValue(), "valid UTF-8 extra_data should decode")
	assert.Equal(t, int64(1741410875), out.GetBlockDateTime().GetSeconds())
	assert.Equal(t, testParentHash, out.GetParentHash().GetValue())
	assert.Equal(t, testStateRoot, out.GetStateRoot().GetValue())
	assert.Equal(t, uint64(786432), out.GetBlobGasUsed().GetValue())
	require.NotNil(t, out.GetExcessBlobGas(), "a zero excess_blob_gas is a value, not NULL")
	assert.Equal(t, uint64(0), out.GetExcessBlobGas().GetValue())
}

func TestRowToProto_PreCancunHasNoBlobGas(t *testing.T) {
	row := &blockRow{
		BlockNumber: 1,
		BlockHash:   testBlockHash,
		ParentHash:  testParentHash,
		StateRoot:   testStateRoot,
	}

	out := blockRowToProto(row)

	assert.Nil(t, out.GetBlobGasUsed())
	assert.Nil(t, out.GetExcessBlobGas())
}

func uint64Ptr(v uint64) *uint64 {
	return &v
}

func TestRowToProto_NonUTF8ExtraData(t *testing.T) {
	row := &blockRow{
		BlockNumber: 1,
		BlockHash:   testBlockHash,
		ExtraData:   "0xfffefd", // not valid UTF-8
	}

	out := blockRowToProto(row)

	assert.Equal(t, "0xfffefd", out.GetExtraData().GetValue())
	assert.Nil(t, out.GetExtraDataString(), "non-UTF-8 extra_data should leave extra_data_string unset")
}

func TestRowToProto_EmptyAuthor(t *testing.T) {
	row := &blockRow{
		BlockNumber: 1,
		BlockHash:   testBlockHash,
	}

	out := blockRowToProto(row)

	assert.Nil(t, out.GetAuthor(), "empty author should be nil/unset")
}

func TestCreateEvent_Chunk(t *testing.T) {
	d := &BlockDeriver{
		clientMeta: &xatu.ClientMeta{Name: "test"},
	}

	rows := []blockRow{
		{BlockNumber: 1, BlockHash: testBlockHash},
		{BlockNumber: 2, BlockHash: testBlockHash},
		{BlockNumber: 3, BlockHash: testBlockHash},
	}

	event, err := d.createEvent(rows)
	require.NoError(t, err)

	assert.Equal(t, xatu.Event_EXECUTION_CANONICAL_BLOCK, event.GetEvent().GetName())
	assert.Len(t, event.GetExecutionCanonicalBlock().GetBlocks(), 3)
	assert.Equal(t, uint64(1), event.GetExecutionCanonicalBlock().GetBlocks()[0].GetBlockNumber())
	assert.Equal(t, "test", event.GetMeta().GetClient().GetName())
	assert.NotEmpty(t, event.GetEvent().GetId())
}
