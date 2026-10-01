package v1

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The types below mirror the EIP-7928 RLP layout so tests can build access
// lists with a precisely known shape.
type (
	testStorageWrite struct {
		Index uint32
		Value *uint256.Int
	}

	testSlotChanges struct {
		Slot    *uint256.Int
		Changes []testStorageWrite
	}

	testBalanceChange struct {
		Index   uint32
		Balance *uint256.Int
	}

	testNonceChange struct {
		Index uint32
		Nonce uint64
	}

	testCodeChange struct {
		Index uint32
		Code  []byte
	}

	testAccountAccess struct {
		Address        common.Address
		StorageChanges []testSlotChanges
		StorageReads   []*uint256.Int
		BalanceChanges []testBalanceChange
		NonceChanges   []testNonceChange
		CodeChanges    []testCodeChange
	}
)

func encodeTestBAL(t *testing.T, accounts []testAccountAccess) []byte {
	t.Helper()

	raw, err := rlp.EncodeToBytes(accounts)
	require.NoError(t, err)

	return raw
}

func summaryCounts(s *BlockAccessListSummary) []uint32 {
	return []uint32{
		s.GetAccountsTouched().GetValue(),
		s.GetStorageSlotsChanged().GetValue(),
		s.GetStorageChanges().GetValue(),
		s.GetStorageReads().GetValue(),
		s.GetBalanceChanges().GetValue(),
		s.GetNonceChanges().GetValue(),
		s.GetCodeChanges().GetValue(),
		s.GetTotalChanges().GetValue(),
	}
}

func TestNewBlockAccessListSummaryFromGloas_AllChangeTypes(t *testing.T) {
	raw := encodeTestBAL(t, []testAccountAccess{
		{
			// Written slots 1 (by two transactions) and 2, read slot 9.
			Address: common.Address{0x01},
			StorageChanges: []testSlotChanges{
				{
					Slot: uint256.NewInt(1),
					Changes: []testStorageWrite{
						{Index: 1, Value: uint256.NewInt(10)},
						{Index: 3, Value: uint256.NewInt(11)},
					},
				},
				{
					Slot:    uint256.NewInt(2),
					Changes: []testStorageWrite{{Index: 2, Value: uint256.NewInt(20)}},
				},
			},
			StorageReads:   []*uint256.Int{uint256.NewInt(9)},
			BalanceChanges: []testBalanceChange{{Index: 1, Balance: uint256.NewInt(100)}, {Index: 2, Balance: uint256.NewInt(90)}},
			NonceChanges:   []testNonceChange{{Index: 1, Nonce: 5}},
		},
		{
			// The same slot numbers on another address are distinct pairs.
			Address: common.Address{0x02},
			StorageChanges: []testSlotChanges{
				{
					Slot:    uint256.NewInt(1),
					Changes: []testStorageWrite{{Index: 1, Value: uint256.NewInt(7)}},
				},
			},
			StorageReads: []*uint256.Int{uint256.NewInt(9), uint256.NewInt(10)},
			CodeChanges:  []testCodeChange{{Index: 4, Code: []byte{0x60, 0x00}}},
		},
		{
			// Touched only: no changes and no reads.
			Address: common.Address{0x03},
		},
	})

	summary, err := NewBlockAccessListSummaryFromGloas(raw)
	require.NoError(t, err)

	assert.Equal(t, []uint32{
		3, // accounts touched
		3, // distinct (address, slot) writes: (1,1), (1,2), (2,1)
		4, // storage write records: 2 + 1 + 1
		3, // distinct (address, slot) reads: (1,9), (2,9), (2,10)
		2, // balance changes
		1, // nonce changes
		1, // code changes
		8, // total changes: 4 + 2 + 1 + 1
	}, summaryCounts(summary))

	assert.Equal(t, uint32(len(raw)), summary.GetBalSizeBytes().GetValue())
	assert.Equal(t, crypto.Keccak256Hash(raw).Hex(), summary.GetBalHash().GetValue())
}

func TestNewBlockAccessListSummaryFromGloas_HashMatchesBlockHeaderHash(t *testing.T) {
	raw := encodeTestBAL(t, []testAccountAccess{
		{Address: common.Address{0x0a}, StorageReads: []*uint256.Int{uint256.NewInt(1)}},
		{Address: common.Address{0x0b}, NonceChanges: []testNonceChange{{Index: 1, Nonce: 2}}},
	})

	var decoded bal.BlockAccessList
	require.NoError(t, rlp.DecodeBytes(raw, &decoded))

	summary, err := NewBlockAccessListSummaryFromGloas(raw)
	require.NoError(t, err)

	assert.Equal(t, decoded.Hash().Hex(), summary.GetBalHash().GetValue())
}

func TestNewBlockAccessListSummaryFromGloas_EmptyList(t *testing.T) {
	raw := encodeTestBAL(t, []testAccountAccess{})
	require.Equal(t, []byte{0xc0}, raw)

	summary, err := NewBlockAccessListSummaryFromGloas(raw)
	require.NoError(t, err)

	assert.Equal(t, make([]uint32, 8), summaryCounts(summary))
	assert.Equal(t, uint32(1), summary.GetBalSizeBytes().GetValue())
	assert.Equal(t, crypto.Keccak256Hash(raw).Hex(), summary.GetBalHash().GetValue())
}

func TestNewBlockAccessListSummaryFromGloas_Undecodable(t *testing.T) {
	for name, raw := range map[string][]byte{
		"nil":         nil,
		"not rlp":     {0xff, 0xfe, 0xfd},
		"wrong shape": {0xc1, 0x01},
	} {
		t.Run(name, func(t *testing.T) {
			summary, err := NewBlockAccessListSummaryFromGloas(raw)

			require.Error(t, err)
			assert.Nil(t, summary)
		})
	}
}

// TestNewBlockAccessListSummaryFromGloas_RealDevnetData uses the access list
// captured from a Gloas devnet beacon node (slot 200): two read-only system
// contracts with four slot reads each, the history storage contract with one
// write and the beacon roots contract with two writes.
func TestNewBlockAccessListSummaryFromGloas_RealDevnetData(t *testing.T) {
	rawHex := "f8d1de9400000961ef480eb55e80d19ad83579a64c007002c0c480010203c0c0c0" +
		"de940000bbddc7ce488642fb579f8b00f3a590007251c0c480010203c0c0c0" +
		"f841940000f90827f1c53a10cb7a02335b175320002935e7e681c7e3e280a02da04ac41c52c5ed22ac6e519822f2a3b5e852d835cab242548ba7cd531b8519c0c0c0c0" +
		"f84e94000f3df6d732807ef1319fb7b8bb8522d0beac02f4cb821316c7c6808469c1e4ede7823315e3e280a03879f24e13e88a5db134e7092a18829d88875fc5c4252d3f27b21b7a96fd2e8bc0c0c0c0"

	raw, err := hex.DecodeString(rawHex)
	require.NoError(t, err)

	summary, err := NewBlockAccessListSummaryFromGloas(raw)
	require.NoError(t, err)

	assert.Equal(t, []uint32{4, 3, 3, 8, 0, 0, 0, 3}, summaryCounts(summary))
	assert.Equal(t, uint32(211), summary.GetBalSizeBytes().GetValue())
	assert.Equal(t, "0x77d73a8c93424f86fec3b88a664fe5a13134e1a017a2d5d96eac421773f4baa0", summary.GetBalHash().GetValue())
}

func TestUint32Count(t *testing.T) {
	assert.Equal(t, uint32(0), uint32Count(-1))
	assert.Equal(t, uint32(0), uint32Count(0))
	assert.Equal(t, uint32(7), uint32Count(7))
	assert.Equal(t, uint32(1<<32-1), uint32Count(1<<32-1))
	assert.Equal(t, uint32(1<<32-1), uint32Count(1<<40))
}
