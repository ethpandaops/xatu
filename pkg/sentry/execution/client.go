package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

// ExecutionClient represents a unified execution client with both WebSocket and RPC capabilities.
type Client struct {
	log    logrus.FieldLogger
	config *Config
	ctx    context.Context //nolint:containedctx // client ctx only.
	cancel context.CancelFunc

	// Connection clients.
	wsClient   *rpc.Client // For subscriptions (WebSocket)
	rpcClient  *rpc.Client // For RPC calls (HTTP)
	httpClient *http.Client

	// Shared state.
	signer        types.Signer
	clientVersion string
	vmu           sync.RWMutex
}

// NewClient creates a new unified execution client.
func NewClient(ctx context.Context, log logrus.FieldLogger, config *Config) (*Client, error) {
	// Validate required configuration.
	if config.WebsocketEnabled && config.WSAddress == "" {
		return nil, fmt.Errorf("WSAddress is required")
	}

	if config.RPCAddress == "" {
		return nil, fmt.Errorf("RPCAddress is required")
	}

	ctx, cancel := context.WithCancel(ctx)

	client := &Client{
		log:        log.WithField("component", "execution/client"),
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
		vmu:        sync.RWMutex{},
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	// Initialize connections.
	var err error

	// Initialize WebSocket client.
	if config.WebsocketEnabled {
		client.wsClient, err = rpc.DialWebsocket(ctx, config.WSAddress, "")
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to dial execution node WebSocket: %w", err)
		}

		client.log.WithField("address", config.WSAddress).Debug("Connected to execution node WS endpoint")
	}

	// Initialize RPC client (required)
	client.rpcClient, err = rpc.DialOptions(
		ctx,
		config.RPCAddress,
		rpc.WithHTTPClient(client.httpClient),
	)
	if err != nil {
		if client.wsClient != nil {
			client.wsClient.Close() // Clean up WebSocket client
		}

		cancel() // Clean up context

		return nil, fmt.Errorf("failed to dial execution node RPC: %w", err)
	}

	client.log.WithField("address", config.RPCAddress).Debug("Connected to execution node RPC endpoint")

	return client, nil
}

// Start starts the execution client.
func (c *Client) Start(ctx context.Context) error {
	// Initialize client version.
	var clientVersion string
	if err := c.rpcClient.CallContext(ctx, &clientVersion, "web3_clientVersion"); err != nil {
		return fmt.Errorf("failed to get client version: %w", err)
	}

	// Cache the client version.
	c.vmu.Lock()
	c.clientVersion = clientVersion
	c.vmu.Unlock()

	// Initialize the signer.
	c.InitSigner(ctx)

	c.log.WithFields(logrus.Fields{
		"client_version": clientVersion,
		"ws_address":     c.config.WSAddress,
		"rpc_address":    c.config.RPCAddress,
	}).Info("Connected to execution client")

	return nil
}

// Stop stops the execution client.
func (c *Client) Stop(ctx context.Context) error {
	c.log.Info("Stopping execution client")

	c.cancel()

	if c.wsClient != nil {
		c.wsClient.Close()
	}

	if c.rpcClient != nil {
		c.rpcClient.Close()
	}

	return nil
}

// GetClientInfo gets the client version info.
func (c *Client) GetClientInfo(ctx context.Context, version *string) error {
	// First check if we already have the version cached.
	c.vmu.RLock()
	cachedVersion := c.clientVersion
	c.vmu.RUnlock()

	if cachedVersion != "" {
		*version = cachedVersion

		return nil
	}

	// If not cached, fetch it and cache it for next time.
	if err := c.rpcClient.CallContext(ctx, version, "web3_clientVersion"); err != nil {
		return err
	}

	// Cache the version.
	c.vmu.Lock()
	c.clientVersion = *version
	c.vmu.Unlock()

	return nil
}

// GetRPCClient provides access to the RPC client directly.
func (c *Client) GetRPCClient() *rpc.Client {
	return c.rpcClient
}

// GetWebSocketClient provides access to the WebSocket client directly.
func (c *Client) GetWebSocketClient() *rpc.Client {
	return c.wsClient
}

// CallContext calls an RPC method with the given context.
func (c *Client) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return c.rpcClient.CallContext(ctx, result, method, args...)
}

// GetSigner returns the signer.
func (c *Client) GetSigner() types.Signer {
	return c.signer
}

// BatchCallContext performs a batch JSON-RPC call for multiple transactions.
func (c *Client) BatchCallContext(ctx context.Context, method string, params []interface{}) ([]json.RawMessage, error) {
	// Prepare batch requests
	reqs := make([]rpc.BatchElem, len(params))
	for i, param := range params {
		reqs[i] = rpc.BatchElem{
			Method: method,
			Args:   []interface{}{param},
			Result: new(json.RawMessage),
		}
	}

	// Execute batch request
	err := c.rpcClient.BatchCallContext(ctx, reqs)
	if err != nil {
		return nil, fmt.Errorf("batch call failed: %w", err)
	}

	// Collect results and check for per-request errors
	results := make([]json.RawMessage, len(reqs))
	for i, req := range reqs {
		if req.Error != nil {
			// Skip individual errors, we'll handle them when processing
			continue
		}

		if req.Result != nil {
			rawMsg := req.Result.(*json.RawMessage)
			if rawMsg != nil && len(*rawMsg) > 0 {
				results[i] = *rawMsg
			}
		}
	}

	return results, nil
}

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

	if err := c.rpcClient.CallContext(ctx, &tx, RPCMethodGetTransactionByHash, hash); err != nil {
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
