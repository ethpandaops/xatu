package frametx

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Transaction 0 of daisugi-devnet-0 execution block 7104, with an empty nonce_keys list.
const devnetFrameTx = "0x06f8ca8429d103d8c02f94e2ed134a13367e23c4343cea3e031431e4fecb41f89cf86b048080c4820bb88080b86000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000000f4c752bdf5bb3172d8eae93675997970281a673eb3f0fe90b5a23dac69753f84ca010380c482c350808080e30280940146c89fbc8642d78d470e553f21befb3ea1d81ac482c35080850332839b5c80c0cd870ae35d630768008477359400c0"

func TestDecodeNonceKeys(t *testing.T) {
	raw := hexutil.MustDecode(devnetFrameTx)

	tx, err := Decode(raw)
	require.NoError(t, err)

	// Expected values are Nethermind's eth_getBlockByNumber fields for this transaction.
	assert.Equal(t, common.HexToHash("0x0ecdddc0fbb1996a06fa616e5bd51a4031b156a5aa15837ae33261d2485e306d"), tx.Hash)
	assert.Equal(t, common.HexToAddress("0xe2ed134a13367e23c4343cea3e031431e4fecb41"), tx.Sender)
	assert.Equal(t, uint64(0x2f), tx.NonceSeq)
	assert.NotNil(t, tx.NonceKeys)
	assert.Empty(t, tx.NonceKeys)
	assert.Equal(t, uint64(0x19258), tx.Gas())
	assert.Equal(t, big.NewInt(0x77359407), tx.GasPrice(big.NewInt(7)))
	// At a base fee above what max_fee covers per unit of gas, the price is max_fee // gas.
	assert.Equal(t, big.NewInt(29754757281), tx.GasPrice(big.NewInt(30_000_000_000)))

	// A transaction on the account nonce leaves nonce_keys out instead of sending an empty list.
	var fields []rlp.RawValue
	require.NoError(t, rlp.DecodeBytes(raw[1:], &fields))

	payload, err := rlp.EncodeToBytes(append(fields[:1:1], fields[2:]...))
	require.NoError(t, err)

	plain, err := Decode(append([]byte{TxType}, payload...))
	require.NoError(t, err)

	assert.Nil(t, plain.NonceKeys)
	assert.Equal(t, tx.NonceSeq, plain.NonceSeq)
	assert.Equal(t, tx.Sender, plain.Sender)
	assert.Equal(t, tx.Frames, plain.Frames)
	assert.Equal(t, tx.Fees, plain.Fees)
	assert.Equal(t, crypto.Keccak256Hash([]byte{TxType}, payload), plain.Hash)
}
