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

// A type 2 transaction of daisugi-devnet-0 execution block 7104, and the frame transaction (type 6) of block 675,
// which uses a keyed nonce and carries a secp256k1 signature.
const (
	devnetDynamicFeeTx = "0x02f8798429d103d804847735940085010642ac07825b25947318bd88bfce048a0cab9bd0a8cd54214a8574918a14f6ccfe33851800000080c001a0a4e516cfd1ffb8329327cde8acfdc1b3b8faf8bcaf0cb56aeb0dea85add41c96a0547c78bf95f47b0d8f42da0f0e144d12d9eb2b2a806c1af44139fdeda1445d89"
	devnetFrameTx      = "0x06f9012b8429d103d8c584ccbd53ec809405a20b6083bd71c58c10cbfa9db8036bf5a8e206f89bce010380c8830186a083017e808080f86b048080c48275308080b860000000000000000000000000000000000000000000000000000000000000001100000000000000000000000000000000000000000000000000000000000000004f0922a927365fec2cf13867145a3de76b3d26385681cd100c9b3d23c3c03d46de028094000000000000000000000000000000000000deadc482c350807b80f85cf85a019405a20b6083bd71c58c10cbfa9db8036bf5a8e20680b84101f9e4f70b43dcfddbd1ce7062d64373a45945d6772d68e28c35ca070c0fef77137420ebe89837279324ab8195934169a2b9f04542de1ec4eff441471a78c959c6cd8762bcad3f9900008477359400c0"
)

func TestPooledTransactionsWithFrameTransactions(t *testing.T) {
	// A blob-carrying frame transaction in the EIP-7594 network wrapper, as PooledTransactions carries it:
	// the devnet frame transaction with one versioned hash as its last field.
	var fields []rlp.RawValue
	require.NoError(t, rlp.DecodeBytes(hexutil.MustDecode(devnetFrameTx)[1:], &fields))

	blobHash := common.HexToHash("0x01aa")
	blobHashes, err := rlp.EncodeToBytes([]common.Hash{blobHash})
	require.NoError(t, err)

	fields[len(fields)-1] = blobHashes

	blobPayload, err := rlp.EncodeToBytes(fields)
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
		signer:  types.LatestSignerForChainID(big.NewInt(0x29d103d8)),
		network: &networks.Network{Name: "daisugi-devnet-0", ID: 0x29d103d8},
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
	assert.Equal(t, "0x37077c661934ede49d96c490e50a511949de213c33e958936067ce16c744601e", dynamicFee.GetHash())
	assert.Equal(t, uint32(2), dynamicFee.GetType().GetValue())

	// Expected values are Nethermind's eth_getBlockByNumber fields for this transaction.
	frame := events[1].GetMeta().GetClient().GetMempoolTransactionV2()
	assert.Equal(t, "0x576438a5eff9cac236779606800c9ed4b172faf3e760557eef527c3739477f38", frame.GetHash())
	assert.Equal(t, uint32(6), frame.GetType().GetValue())
	assert.Equal(t, common.HexToAddress("0x05a20b6083bd71c58c10cbfa9db8036bf5a8e206").Hex(), frame.GetFrom())
	assert.Equal(t, uint64(0), frame.GetNonce().GetValue())
	assert.Equal(t, uint64(0x43da0), frame.GetGas().GetValue())
	assert.Equal(t, "2000000000", frame.GetGasTipCap())
	assert.Empty(t, frame.GetGasPrice())
	assert.Empty(t, frame.GetGasFeeCap())
	assert.Empty(t, frame.GetTo())
	assert.Equal(t, "303", frame.GetSize())
	assert.Equal(t, devnetFrameTx, events[1].GetMempoolTransactionV2())

	withBlob := events[2].GetMeta().GetClient().GetMempoolTransactionV2()
	assert.Equal(t, crypto.Keccak256Hash([]byte{frametx.TxType}, blobPayload).Hex(), withBlob.GetHash())
	assert.Equal(t, []string{blobHash.Hex()}, withBlob.GetBlobHashes())
	assert.Equal(t, uint64(131072), withBlob.GetBlobGas().GetValue())
	assert.Equal(t, "131072", withBlob.GetBlobSidecarsSize())
}
