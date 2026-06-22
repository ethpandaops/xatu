package execution

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const testBlockHash = "0xfc429da12e414c0e8348fd9bb760b1a118b4dcf73419f021d4ccc2c3b793e05c"

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
