package execution

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	eth "github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/internal/frametx"
	"github.com/ethpandaops/xatu/pkg/mimicry/p2p/handler"
	"github.com/ethpandaops/xatu/pkg/networks"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// Transactions 0x8d (type 2) and 0x1f (frame, type 6) of bogota-devnet-2 execution block 185501.
const (
	devnetDynamicFeeTx = "0x02f8698501a5cdc3058293490e0e825208944d713ded9ad1cf80e4a439f168fd79e8504536e28080c001a06501dfe3910a599a5d92368f8d64f9ce72d675a31ff436b8f3e8cebe881bb7e6a0419df4095d204b3f719ba942313621a1c3fe7e372208b600dba89d1ea1468ae5"
	devnetFrameTx      = "0x06f8e18501a5cdc305c08301717f9487f4e342fa4a02ba22d2cceaf889e74897d464cef866ca010380c4821388808080de02049420a518a530f6c882288b7eb134d5a67a02e7908cc4827530808080dc02049420a518a530f6c882288b7eb134d5a67a02e7908cc201808080de02809420a518a530f6c882288b7eb134d5a67a02e7908cc4827530808080f848f846018080b84101257063ab5e3dc0bbbb13cd3a82121e324232051af202968a1686a0e5f4b028eb32623a330d787d0a9160a7a26a98c2b968960036daa5040278014fd03edd0ef3cd8705e28155b2e8008477359400c0"
)

func TestPooledTransactionsWithFrameTransactions(t *testing.T) {
	// A blob-carrying frame transaction in the EIP-7594 network wrapper, as PooledTransactions carries it.
	blobFrame, err := frametx.Decode(hexutil.MustDecode(devnetFrameTx))
	require.NoError(t, err)

	blobHash := common.HexToHash("0x01aa")
	blobFrame.BlobVersionedHashes = []common.Hash{blobHash}

	blobPayload, err := rlp.EncodeToBytes(blobFrame)
	require.NoError(t, err)

	blob := make([]byte, 131072)
	blob[0] = 1

	wrapped, err := rlp.EncodeToBytes([]any{rlp.RawValue(blobPayload), uint8(1), [][]byte{blob}, [][]byte{{}}, [][]byte{}})
	require.NoError(t, err)

	var list rlp.RawList[*types.Transaction]

	for _, envelope := range [][]byte{
		hexutil.MustDecode(devnetDynamicFeeTx),
		hexutil.MustDecode(devnetFrameTx),
		{0x7f, 0xc0}, // a transaction type nothing decodes must drop only itself
		append([]byte{frametx.TxType}, wrapped...),
	} {
		element, encodeErr := rlp.EncodeToBytes(envelope)
		require.NoError(t, encodeErr)
		require.NoError(t, list.AppendRaw(element))
	}

	// Round-trip through the wire encoding, as the devp2p client receives it.
	packetBytes, err := rlp.EncodeToBytes(&eth.PooledTransactionsPacket{RequestId: 1, List: list})
	require.NoError(t, err)

	var packet eth.PooledTransactionsPacket
	require.NoError(t, rlp.DecodeBytes(packetBytes, &packet))

	p := &Peer{
		log:     logrus.New(),
		signer:  types.LatestSignerForChainID(big.NewInt(0x1a5cdc305)),
		network: &networks.Network{Name: "bogota-devnet-2", ID: 0x1a5cdc305},
		handlers: &handler.Peer{
			CreateNewClientMeta: func(_ context.Context) (*xatu.ClientMeta, error) {
				return &xatu.ClientMeta{Name: "test-mimicry"}, nil
			},
		},
	}

	transactions := p.decodeTransactions(context.Background(), &packet.List)
	require.Len(t, transactions, 3)

	events := make([]*xatu.DecoratedEvent, 0, len(transactions))

	for _, tx := range transactions {
		event, err := p.handleTransaction(context.Background(), time.Now(), tx)
		require.NoError(t, err)

		events = append(events, event)
	}

	dynamicFee := events[0].GetMeta().GetClient().GetMempoolTransactionV2()
	assert.Equal(t, "0xd5fdc36e1076bc05ff7b397aeaece0395d5005236413fca57b88f8001e35f987", dynamicFee.GetHash())
	assert.Equal(t, uint32(2), dynamicFee.GetType().GetValue())

	// Expected values are the EL's eth_getBlockByNumber fields for this transaction.
	frame := events[1].GetMeta().GetClient().GetMempoolTransactionV2()
	assert.Equal(t, "0xf855ff365dc91ce3c2c928c49a6854cb31d177ea3b7f12145aa3b8468a204bcc", frame.GetHash())
	assert.Equal(t, uint32(6), frame.GetType().GetValue())
	assert.Equal(t, common.HexToAddress("0x87f4e342fa4a02ba22d2cceaf889e74897d464ce").Hex(), frame.GetFrom())
	assert.Equal(t, uint64(0x1717f), frame.GetNonce().GetValue())
	assert.Equal(t, uint64(0x14385), frame.GetGas().GetValue())
	assert.Equal(t, "2000000000", frame.GetGasTipCap())
	assert.Empty(t, frame.GetGasPrice())
	assert.Empty(t, frame.GetGasFeeCap())
	assert.Empty(t, frame.GetTo())
	assert.Equal(t, "228", frame.GetSize())
	assert.Equal(t, devnetFrameTx, events[1].GetMempoolTransactionV2())

	withBlob := events[2].GetMeta().GetClient().GetMempoolTransactionV2()
	assert.Equal(t, crypto.Keccak256Hash([]byte{frametx.TxType}, blobPayload).Hex(), withBlob.GetHash())
	assert.Equal(t, []string{blobHash.Hex()}, withBlob.GetBlobHashes())
	assert.Equal(t, uint64(131072), withBlob.GetBlobGas().GetValue())
	assert.Equal(t, "131072", withBlob.GetBlobSidecarsSize())
}
