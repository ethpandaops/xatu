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
	"github.com/ethpandaops/ethcore/pkg/ethereum/clients"
	"github.com/sirupsen/logrus"
)

//go:generate mockgen -package mock -destination mock/client.mock.go github.com/ethpandaops/xatu/pkg/sentry/execution ClientProvider

// ClientProvider defines the interface for unified execution client (WS and RPC) operations.
type ClientProvider interface {
	// GetTxpoolContent retrieves the full transaction pool content.
	GetTxpoolContent(ctx context.Context) (json.RawMessage, error)

	// GetPendingTransactions retrieves pending transactions.
	GetPendingTransactions(ctx context.Context) ([]json.RawMessage, error)

	// BatchGetTransactionsByHash retrieves transactions by their hashes.
	BatchGetTransactionsByHash(ctx context.Context, hashes []string) ([]json.RawMessage, error)

	// SubscribeToNewPendingTxs subscribes to new pending transaction notifications.
	SubscribeToNewPendingTxs(ctx context.Context) (<-chan string, <-chan error, error)

	// CallContext performs a JSON-RPC call with the given arguments.
	CallContext(ctx context.Context, result any, method string, args ...any) error
}

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
	signer                    types.Signer
	clientVersion             string
	clientImplementation      string
	clientVersionParsed       string
	clientVersionMajor        string
	clientVersionMinor        string
	clientVersionPatch        string
	clientMetadataInitialized bool
	vmu                       sync.RWMutex
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
	// Initialize client version and metadata.
	if err := c.refreshClientMetadata(ctx); err != nil {
		return fmt.Errorf("failed to get client version: %w", err)
	}

	// Initialize the signer.
	c.InitSigner(ctx)

	c.log.WithFields(logrus.Fields{
		"client_version":        c.clientVersion,
		"client_implementation": c.clientImplementation,
		"ws_address":            c.config.WSAddress,
		"rpc_address":           c.config.RPCAddress,
	}).Info("Connected to execution client")

	// Start periodic refresh of client metadata (every 5 minutes).
	go c.startPeriodicMetadataRefresh()

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

// GetTxpoolContent retrieves the full transaction pool content.
func (c *Client) GetTxpoolContent(ctx context.Context) (json.RawMessage, error) {
	var result json.RawMessage
	err := c.CallContext(ctx, &result, RPCMethodTxpoolContent)

	return result, err
}

// GetPendingTransactions retrieves pending transactions.
func (c *Client) GetPendingTransactions(ctx context.Context) ([]json.RawMessage, error) {
	var result json.RawMessage
	if err := c.CallContext(ctx, &result, RPCMethodPendingTransactions); err != nil {
		return nil, err
	}

	// Parse into array of raw messages.
	var txs []json.RawMessage
	if err := json.Unmarshal(result, &txs); err != nil {
		return nil, err
	}

	return txs, nil
}

// BatchGetTransactionsByHash retrieves transactions by their hashes.
func (c *Client) BatchGetTransactionsByHash(ctx context.Context, hashes []string) ([]json.RawMessage, error) {
	params := make([]any, len(hashes))
	for i, hash := range hashes {
		params[i] = hash
	}

	return c.BatchCallContext(ctx, RPCMethodGetTransactionByHash, params)
}

// SubscribeToNewPendingTxs subscribes to new pending transaction notifications
//
//nolint:gocritic // No need for named returns.
func (c *Client) SubscribeToNewPendingTxs(ctx context.Context) (<-chan string, <-chan error, error) {
	if c.wsClient == nil {
		return nil, nil, fmt.Errorf("websocket client not initialized")
	}

	txChan := make(chan string)

	sub, err := c.wsClient.EthSubscribe(ctx, txChan, string(SubNewPendingTransactions))
	if err != nil {
		return nil, nil, err
	}

	return txChan, sub.Err(), nil
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
func (c *Client) CallContext(ctx context.Context, result any, method string, args ...any) error {
	return c.rpcClient.CallContext(ctx, result, method, args...)
}

// GetSigner returns the signer.
func (c *Client) GetSigner() types.Signer {
	return c.signer
}

// BatchCallContext performs a batch JSON-RPC call for multiple transactions.
func (c *Client) BatchCallContext(ctx context.Context, method string, params []any) ([]json.RawMessage, error) {
	// Prepare batch requests.
	reqs := make([]rpc.BatchElem, len(params))
	for i, param := range params {
		reqs[i] = rpc.BatchElem{
			Method: method,
			Args:   []any{param},
			Result: new(json.RawMessage),
		}
	}

	// Execute batch request.
	err := c.rpcClient.BatchCallContext(ctx, reqs)
	if err != nil {
		return nil, fmt.Errorf("batch call failed: %w", err)
	}

	// Collect results and check for per-request errors.
	results := make([]json.RawMessage, len(reqs))

	for i, req := range reqs {
		// Skip individual errors, we'll handle them when processing.
		if req.Error != nil {
			continue
		}

		if req.Result != nil {
			var ok bool

			rawMsg, ok := req.Result.(*json.RawMessage)
			if !ok {
				continue
			}

			if rawMsg != nil && len(*rawMsg) > 0 {
				results[i] = *rawMsg
			}
		}
	}

	return results, nil
}

// hexToDecimalString converts a hexadecimal string (with or without 0x prefix) to a decimal string.
// Returns an error if the hex string is invalid.
func hexToDecimalString(hexStr string) (string, error) {
	if hexStr == "" {
		return "0", nil
	}

	// Parse the hex string as a big integer.
	value, ok := new(big.Int).SetString(hexStr, 0)
	if !ok {
		return "", fmt.Errorf("invalid hex string: %s", hexStr)
	}

	return value.String(), nil
}

// DebugStateSize retrieves the state size from the execution client.
// blockNumber can be "latest", "earliest", "pending", or a specific block number (e.g., "0x1234").
// If empty, defaults to "latest".
func (c *Client) DebugStateSize(ctx context.Context, blockNumber string) (*DebugStateSizeResponse, error) {
	if blockNumber == "" {
		blockNumber = "latest"
	}

	var result DebugStateSizeResponse

	err := c.rpcClient.CallContext(ctx, &result, RPCMethodDebugStateSize, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("debug_stateSize call failed: %w", err)
	}

	// Convert all hex fields to decimal strings.
	result.AccountBytes, err = hexToDecimalString(result.AccountBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AccountBytes: %w", err)
	}

	result.AccountTrienodeBytes, err = hexToDecimalString(result.AccountTrienodeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AccountTrienodeBytes: %w", err)
	}

	result.AccountTrienodes, err = hexToDecimalString(result.AccountTrienodes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert AccountTrienodes: %w", err)
	}

	result.Accounts, err = hexToDecimalString(result.Accounts)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Accounts: %w", err)
	}

	result.BlockNumber, err = hexToDecimalString(result.BlockNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to convert BlockNumber: %w", err)
	}

	result.ContractCodeBytes, err = hexToDecimalString(result.ContractCodeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ContractCodeBytes: %w", err)
	}

	result.ContractCodes, err = hexToDecimalString(result.ContractCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert ContractCodes: %w", err)
	}

	result.StorageBytes, err = hexToDecimalString(result.StorageBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert StorageBytes: %w", err)
	}

	result.StorageTrienodeBytes, err = hexToDecimalString(result.StorageTrienodeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert StorageTrienodeBytes: %w", err)
	}

	result.StorageTrienodes, err = hexToDecimalString(result.StorageTrienodes)
	if err != nil {
		return nil, fmt.Errorf("failed to convert StorageTrienodes: %w", err)
	}

	result.Storages, err = hexToDecimalString(result.Storages)
	if err != nil {
		return nil, fmt.Errorf("failed to convert Storages: %w", err)
	}

	return &result, nil
}

// SubscribeToNewHeads subscribes to new block header notifications via WebSocket.
// Returns a channel for receiving block headers and a channel for errors.
// Requires WebSocket to be enabled in the client configuration.
func (c *Client) SubscribeToNewHeads(ctx context.Context) (headerChan <-chan *types.Header, errChan <-chan error, err error) {
	if c.wsClient == nil {
		return nil, nil, fmt.Errorf("WebSocket not enabled")
	}

	hChan := make(chan *types.Header, 100)
	eChan := make(chan error, 1)

	sub, subErr := c.wsClient.EthSubscribe(ctx, hChan, "newHeads")
	if subErr != nil {
		return nil, nil, fmt.Errorf("failed to subscribe to new heads: %w", subErr)
	}

	// Forward subscription errors to error channel.
	go func() {
		defer close(hChan)
		defer close(eChan)

		select {
		case subErr := <-sub.Err():
			if subErr != nil {
				eChan <- subErr
			}
		case <-ctx.Done():
			sub.Unsubscribe()
		}
	}()

	return hChan, eChan, nil
}

// GetSender retrieves the sender of a transaction.
func (c *Client) GetSender(tx *types.Transaction) (common.Address, error) {
	if c.signer == nil {
		return common.Address{}, fmt.Errorf("signer not initialized")
	}

	return c.signer.Sender(tx)
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

// refreshClientMetadata fetches and parses the execution client version information.
func (c *Client) refreshClientMetadata(ctx context.Context) error {
	var clientVersion string
	if err := c.rpcClient.CallContext(ctx, &clientVersion, "web3_clientVersion"); err != nil {
		return err
	}

	// Parse the client version string using ethcore's shared parser.
	implementation, version, versionMajor, versionMinor, versionPatch := clients.ParseExecutionClientVersion(clientVersion)

	c.vmu.Lock()
	c.clientVersion = clientVersion
	c.clientImplementation = implementation
	c.clientVersionParsed = version
	c.clientVersionMajor = versionMajor
	c.clientVersionMinor = versionMinor
	c.clientVersionPatch = versionPatch
	c.clientMetadataInitialized = true
	c.vmu.Unlock()

	return nil
}

// startPeriodicMetadataRefresh periodically refreshes the client metadata.
func (c *Client) startPeriodicMetadataRefresh() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.refreshClientMetadata(c.ctx); err != nil {
				c.log.WithError(err).Warn("Failed to refresh execution client metadata")
			}
		}
	}
}

// ClientMetadata contains parsed execution client version information.
type ClientMetadata struct {
	Implementation string
	Version        string
	VersionMajor   string
	VersionMinor   string
	VersionPatch   string
	Initialized    bool
}

// GetClientMetadata returns the cached execution client metadata.
// Returns empty strings if metadata hasn't been initialized yet.
func (c *Client) GetClientMetadata() ClientMetadata {
	c.vmu.RLock()
	defer c.vmu.RUnlock()

	return ClientMetadata{
		Implementation: c.clientImplementation,
		Version:        c.clientVersionParsed,
		VersionMajor:   c.clientVersionMajor,
		VersionMinor:   c.clientVersionMinor,
		VersionPatch:   c.clientVersionPatch,
		Initialized:    c.clientMetadataInitialized,
	}
}
