package execution

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// GetSender retrieves the sender of a transaction.
func (c *Client) GetSender(tx *types.Transaction) (common.Address, error) {
	if c.signer == nil {
		return common.Address{}, fmt.Errorf("signer not initialized")
	}

	return c.signer.Sender(tx)
}

// GetTransactionByHash retrieves a transaction by its hash.
func (c *Client) GetTransactionByHash(ctx context.Context, hash string) (*types.Transaction, error) {
	var tx *types.Transaction

	if err := c.rpcClient.CallContext(ctx, &tx, "eth_getTransactionByHash", hash); err != nil {
		return nil, fmt.Errorf("failed to get transaction: %w", err)
	}

	if tx == nil {
		return nil, fmt.Errorf("transaction not found: %s", hash)
	}

	return tx, nil
}

// GetRawTransactionByHash retrieves a raw transaction by its hash.
func (c *Client) GetRawTransactionByHash(ctx context.Context, hash string) (string, error) {
	var rawTx string

	// Some clients might not support eth_getRawTransactionByHash, fallback to getting
	// the transaction and encoding it.
	if err := c.rpcClient.CallContext(ctx, &rawTx, "eth_getRawTransactionByHash", hash); err != nil {
		tx, err := c.GetTransactionByHash(ctx, hash)
		if err != nil {
			return "", err
		}

		txBytes, err := tx.MarshalBinary()
		if err != nil {
			return "", fmt.Errorf("failed to marshal transaction: %w", err)
		}

		rawTx = hexutil.Encode(txBytes)
	}

	return rawTx, nil
}

// InitSigner initialises the transaction signer. This is used to determine mempool tx senders.
func (c *Client) InitSigner(ctx context.Context) {
	var (
		chainIDHex string
		chainID    = params.MainnetChainConfig.ChainID.Uint64()
	)

	// Get chain ID and initialise our signer.
	if err := c.rpcClient.CallContext(ctx, &chainIDHex, "eth_chainId"); err != nil {
		c.log.WithError(err).Warn("Failed to get chain ID, using mainnet as default")
	} else if decoded, err := hexutil.DecodeUint64(chainIDHex); err != nil {
		c.log.WithError(err).Warn("Failed to decode chain ID, using mainnet as default")
	} else {
		chainID = decoded
	}

	chainIDInt := new(big.Int).SetUint64(chainID)
	c.signer = types.NewCancunSigner(chainIDInt)
}
