package sentry

import (
	"context"
	"encoding/json"
	"io"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	ttlcache "github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	execEvent "github.com/ethpandaops/xatu/pkg/sentry/event/execution"
)

// TestMempoolTransactionKeepsSignedIdentity checks that a transaction read from txpool_content is recorded with the
// hash, type and blob fields of the signed transaction the execution client holds.
func TestMempoolTransactionKeepsSignedIdentity(t *testing.T) {
	key, err := crypto.GenerateKey()
	require.NoError(t, err)

	var (
		chainID = big.NewInt(1)
		signer  = types.LatestSignerForChainID(chainID)
		to      = common.HexToAddress("0x00000000000000000000000000000000000000aa")
		blob    = common.HexToHash("0x0100000000000000000000000000000000000000000000000000000000000001")
	)

	txs := map[string]types.TxData{
		"dynamic fee": &types.DynamicFeeTx{
			ChainID: chainID, Nonce: 7, GasTipCap: big.NewInt(2), GasFeeCap: big.NewInt(30), Gas: 21000, To: &to,
			Value: big.NewInt(5),
		},
		// Pooled blob transactions come over JSON-RPC without their sidecar.
		"blob": &types.BlobTx{
			ChainID: uint256.NewInt(1), Nonce: 8, GasTipCap: uint256.NewInt(2), GasFeeCap: uint256.NewInt(30),
			Gas: 21000, To: to, Value: uint256.NewInt(0), BlobFeeCap: uint256.NewInt(9), BlobHashes: []common.Hash{blob},
		},
	}

	for name, data := range txs {
		t.Run(name, func(t *testing.T) {
			signed, err := types.SignNewTx(key, signer, data)
			require.NoError(t, err)

			// txpool_content returns the signed transaction's fields plus its sender.
			raw, err := signed.MarshalJSON()
			require.NoError(t, err)

			var fields map[string]any
			require.NoError(t, json.Unmarshal(raw, &fields))

			sender := crypto.PubkeyToAddress(key.PublicKey).Hex()
			fields["from"] = sender

			txData, err := json.Marshal(fields)
			require.NoError(t, err)

			parsed, err := parseRawTransactionFromTxPool(txData, signed.Hash())
			require.NoError(t, err)

			// A transaction that decodes to a different hash than the client reported is rejected, not recorded.
			_, err = parseRawTransactionFromTxPool(txData, common.HexToHash("0x01"))
			require.Error(t, err)

			logger := logrus.New()
			logger.SetOutput(io.Discard)

			meta := &xatu.ClientMeta{AdditionalData: &xatu.ClientMeta_MempoolTransactionV2{
				MempoolTransactionV2: &xatu.ClientMeta_AdditionalMempoolTransactionV2Data{Hash: signed.Hash().Hex(), From: sender},
			}}

			event, err := execEvent.NewMempoolTransaction(
				logger, parsed, time.Now(), ttlcache.New[string, time.Time](), meta, signer,
			).Decorate(context.Background())
			require.NoError(t, err)

			got := event.GetMeta().GetClient().GetMempoolTransactionV2()
			require.NotNil(t, got)

			assert.Equal(t, signed.Hash().Hex(), got.GetHash())
			assert.Equal(t, signed.Hash(), parsed.Hash(), "the decoded transaction is the signed one")
			assert.Equal(t, uint32(signed.Type()), got.GetType().GetValue())
			assert.Equal(t, sender, got.GetFrom())
			assert.Equal(t, signed.Nonce(), got.GetNonce().GetValue())
			assert.Equal(t, signed.GasTipCap().String(), got.GetGasTipCap())

			if signed.Type() == types.BlobTxType {
				assert.Equal(t, []string{blob.Hex()}, got.GetBlobHashes())
				assert.Equal(t, "9", got.GetBlobGasFeeCap())
				assert.Empty(t, got.GetBlobSidecarsSize(), "no sidecar was attached")
			}
		})
	}
}
