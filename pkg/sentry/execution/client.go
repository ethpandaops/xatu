package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

// Client represents an execution client.
type Client struct {
	log           logrus.FieldLogger
	config        *Config
	ctx           context.Context //nolint:containedctx // client ctx only.
	cancel        context.CancelFunc
	rpcClient     *rpc.Client
	httpClient    *http.Client
	signer        types.Signer
	callbacks     map[SubscriptionType][]EventCallback
	callbacksLock sync.RWMutex
	polling       bool

	// WaitGroup for in-flight events, allows us to shutdown gracefully (hopefully
	// without dropping any events).
	wg sync.WaitGroup

	// We cache the client version. Each event is hydrated with client metadata, we
	// don't need to be hitting the EL for this everytime.
	clientVersion string
	vmu           sync.RWMutex
}

// NewClient creates a new execution client.
func NewClient(ctx context.Context, log logrus.FieldLogger, config *Config) (*Client, error) {
	ctx, cancel := context.WithCancel(ctx)

	var (
		rpcClientent *rpc.Client
		err          error
		client       = &Client{
			log:           log.WithField("module", "sentry/execution"),
			config:        config,
			ctx:           ctx,
			cancel:        cancel,
			wg:            sync.WaitGroup{},
			clientVersion: "",
			vmu:           sync.RWMutex{},
			callbacks:     make(map[SubscriptionType][]EventCallback),
			httpClient: &http.Client{
				Timeout: 30 * time.Second,
			},
		}
	)

	// Depending on our execution address URL scheme, we'll either use a websocket or http client.
	client.polling = !strings.HasPrefix(config.Address, "ws://") && !strings.HasPrefix(config.Address, "wss://")

	switch client.polling {
	case false:
		rpcClientent, err = rpc.DialWebsocket(ctx, config.Address, "")
	default:
		rpcClientent, err = rpc.DialOptions(
			ctx,
			config.Address,
			rpc.WithHTTPClient(client.httpClient),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to dial execution node RPC: %w", err)
	}

	client.rpcClient = rpcClientent

	return client, nil
}

// Start starts the execution client.
func (c *Client) Start(ctx context.Context) error {
	var clientVersion string
	if err := c.rpcClient.CallContext(ctx, &clientVersion, "web3_clientVersion"); err != nil {
		return fmt.Errorf("failed to get client version: %w", err)
	}

	// Cache the client version, we don't need to be looking this up every event when
	// building the metadata. It won't change.
	c.vmu.Lock()
	c.clientVersion = clientVersion
	c.vmu.Unlock()

	// Initialise the signer. This is used to determine mempool tx senders.
	c.InitSigner(ctx)

	c.log.WithFields(logrus.Fields{
		"client_version": clientVersion,
		"polling":        c.polling,
		"address":        c.config.Address,
	}).Info("Connected to execution client")

	if c.polling {
		c.startPolling()
	} else {
		c.startSubscriptions(ctx)
	}

	return nil
}

// Stop stops the execution client.
func (c *Client) Stop(ctx context.Context) error {
	c.log.Info("Stopping execution client")

	c.cancel()

	// Wait for goroutines to finish.
	c.wg.Wait()

	c.rpcClient.Close()

	return nil
}

// Subscribe adds a callback for a specific subscription type.
// This is intended for runtime callbacks added after initialisation.
func (c *Client) Subscribe(subType SubscriptionType, callback EventCallback) {
	c.callbacksLock.Lock()
	defer c.callbacksLock.Unlock()

	c.callbacks[subType] = append(c.callbacks[subType], callback)
	c.log.WithField("subscription_type", subType).Debug("Added runtime subscription callback")
}

// OnNewPendingTransactions registers a callback for when a new pending transaction is received.
func (c *Client) OnNewPendingTransactions(callback TransactionCallback) {
	c.Subscribe(SubNewPendingTransactions, func(ctx context.Context, event interface{}) error {
		txHash, ok := event.(string)
		if !ok {
			return fmt.Errorf("expected string event, got %T", event)
		}

		return callback(ctx, txHash)
	})
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

// startSubscriptions starts all registered subscriptions.
func (c *Client) startSubscriptions(ctx context.Context) {
	// Get all subscription types with registered callbacks.
	subTypes := c.getRegisteredSubscriptionTypes()

	// If no callbacks are registered, default to newPendingTransactions.
	if len(subTypes) == 0 {
		subTypes = []SubscriptionType{SubNewPendingTransactions}
	}

	// Start each subscription.
	for _, subType := range subTypes {
		c.startSubscription(ctx, subType)
	}

	c.log.WithField("topics", subTypes).Info("Subscribing to events upstream")
}

// startSubscription starts a subscription of the specified type.
func (c *Client) startSubscription(ctx context.Context, subType SubscriptionType) {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		for {
			select {
			case <-c.ctx.Done():
				return
			default:
				c.subscribeToEvent(ctx, subType)

				// Let it catch its breath before reconnecting.
				time.Sleep(5 * time.Second)
			}
		}
	}()
}

// subscribeToEvent subscribes to a specific event type.
func (c *Client) subscribeToEvent(ctx context.Context, subType SubscriptionType) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		sub        *rpc.ClientSubscription
		err        error
		handleFunc func(interface{}) error
	)

	var (
		statsTicker = time.NewTicker(c.config.PollingInterval)
		txCounter   = 0
		txCounterMu sync.Mutex
	)

	defer statsTicker.Stop()

	switch subType {
	case SubNewPendingTransactions:
		// As we're connecting to the EL via websocket, we need a way to periodically
		// log the number of transactions we're receiving. This is not a problem when
		// polling via RPC.
		c.wg.Add(1)

		go func() {
			defer c.wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case <-statsTicker.C:
					txCounterMu.Lock()
					count := txCounter
					txCounter = 0 // Reset counter
					txCounterMu.Unlock()

					if count > 0 {
						c.log.WithFields(logrus.Fields{
							"method":   "websocket_subscribe",
							"tx_count": count,
						}).Info("Found new pending transactions in mempool")
					}
				}
			}
		}()

		txChan := make(chan string)

		sub, err = c.rpcClient.EthSubscribe(ctx, txChan, string(subType))
		if err != nil {
			c.log.WithError(err).WithField("subscription_type", subType).Error("Failed to subscribe")

			return
		}

		handleFunc = func(event interface{}) error {
			// Get the transaction from the channel.
			txHash := <-txChan

			// Increment counter for stats.
			txCounterMu.Lock()
			txCounter++
			txCounterMu.Unlock()

			// Finally, process it.
			return c.handleEvent(ctx, subType, txHash)
		}

	default:
		c.log.WithField("subscription_type", subType).Error("Unsupported subscription type")

		return
	}

	defer sub.Unsubscribe()

	for {
		select {
		case err := <-sub.Err():
			c.log.WithError(err).WithField("subscription_type", subType).Error("Subscription error")

			return
		case <-ctx.Done():
			return
		default:
			if err := handleFunc(nil); err != nil {
				c.log.WithError(err).WithField("subscription_type", subType).Error("Error handling event")
			}
		}
	}
}

// startPolling starts polling for supported subscription types.
func (c *Client) startPolling() {
	// Get all subscription types with registered callbacks.
	subTypes := c.getRegisteredSubscriptionTypes()

	// If no callbacks are registered, default to newPendingTransactions.
	if len(subTypes) == 0 {
		subTypes = []SubscriptionType{SubNewPendingTransactions}
	}

	// Start polling for each supported type.
	for _, subType := range subTypes {
		c.pollForEvents(subType)
	}
}

// pollForEvents starts polling for events of a specific type. We're supporting multiple different
// polling methods here, due to client support of the different methods.
func (c *Client) pollForEvents(subType SubscriptionType) {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()

		ticker := time.NewTicker(c.config.PollingInterval)
		defer ticker.Stop()

		previousHashes := make(map[string]bool)

		// Determine supported polling methods for this subscription type.
		var pollingMethod string

		switch subType {
		case SubNewPendingTransactions:
			// Determine the best polling method for pending transactions.
			pollingMethod = c.determinePendingTxPollingMethod()
		default:
			c.log.WithField("subscription_type", subType).Error("No polling method available for this subscription type")

			return
		}

		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				if subType == SubNewPendingTransactions {
					switch pollingMethod {
					case RPCMethodTxpoolContent:
						c.pollPendingTransactionsTxpool(previousHashes)
					case RPCMethodPendingTransactions:
						c.pollPendingTransactionsEth(previousHashes)
					default:
						c.log.WithField("polling_method", pollingMethod).Error("Unknown polling method")
					}
				}

				// Cleanup old hashes periodically.
				if len(previousHashes) > 10000 {
					c.log.Info("Clearing transaction hash cache (exceeded 10000 entries)")

					previousHashes = make(map[string]bool)
				}
			}
		}
	}()
}

// determinePendingTxPollingMethod determines the best method to poll for pending transactions.
func (c *Client) determinePendingTxPollingMethod() string {
	// Try txpool_content first (preferred method).
	var txpoolContentResult struct {
		Pending map[string]map[string]json.RawMessage `json:"pending"`
	}

	txpoolErr := c.rpcClient.CallContext(c.ctx, &txpoolContentResult, RPCMethodTxpoolContent)
	if txpoolErr == nil {
		return RPCMethodTxpoolContent
	}

	// Try eth_pendingTransactions as fallback.
	var pendingTxs []string

	pendingTxErr := c.rpcClient.CallContext(c.ctx, &pendingTxs, RPCMethodPendingTransactions)
	if pendingTxErr == nil {
		return RPCMethodPendingTransactions
	}

	// If neither method works, log a warning and default to eth_pendingTransactions.
	c.log.Warnf(
		"Node doesn't support %s or %s, transaction polling may not work",
		RPCMethodTxpoolContent,
		RPCMethodPendingTransactions,
	)

	return RPCMethodPendingTransactions
}

// pollPendingTransactionsTxpool polls for pending transactions using the txpool_content method.
func (c *Client) pollPendingTransactionsTxpool(previousHashes map[string]bool) {
	var result struct {
		Pending map[string]map[string]json.RawMessage `json:"pending"`
	}

	if err := c.rpcClient.CallContext(c.ctx, &result, RPCMethodTxpoolContent); err != nil {
		c.log.WithError(err).Error("Failed to poll txpool content")

		return
	}

	var (
		txCount int
		unseen  = make(map[string]bool)
	)

	for _, accounts := range result.Pending {
		txCount += len(accounts)
	}

	// Process pending transactions.
	for _, accounts := range result.Pending {
		for nonce, txData := range accounts {
			// Extract the hash from the transaction data.
			var tx struct {
				Hash string `json:"hash"`
			}

			if err := json.Unmarshal(txData, &tx); err != nil {
				c.log.WithError(err).WithField("nonce", nonce).Error("Failed to parse transaction data")

				continue
			}

			txHash := tx.Hash

			if previousHashes[txHash] {
				continue
			}

			previousHashes[txHash] = true
			unseen[txHash] = true

			if err := c.handleEvent(c.ctx, SubNewPendingTransactions, txHash); err != nil {
				c.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to handle event")
			}
		}
	}

	if len(unseen) > 0 {
		c.log.WithFields(logrus.Fields{
			"method":             RPCMethodTxpoolContent,
			"total_pending_txs":  txCount,
			"unseen_pending_txs": len(unseen),
		}).Info("Found new pending transactions in mempool")
	}
}

// pollPendingTransactionsEth polls for pending transactions using the eth_pendingTransactions method.
func (c *Client) pollPendingTransactionsEth(previousHashes map[string]bool) {
	var result []string
	if err := c.rpcClient.CallContext(c.ctx, &result, RPCMethodPendingTransactions); err != nil {
		c.log.WithError(err).Error("Failed to poll pending transactions")

		return
	}

	unseen := make(map[string]bool)

	for _, txHash := range result {
		if previousHashes[txHash] {
			continue
		}

		previousHashes[txHash] = true
		unseen[txHash] = true

		if err := c.handleEvent(c.ctx, SubNewPendingTransactions, txHash); err != nil {
			c.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to handle event")
		}
	}

	if len(unseen) > 0 {
		c.log.WithFields(logrus.Fields{
			"method":             RPCMethodPendingTransactions,
			"total_pending_txs":  len(result),
			"unseen_pending_txs": len(unseen),
		}).Info("Found new pending transactions in mempool")
	}
}

// handleEvent routes an event to the appropriate callbacks.
func (c *Client) handleEvent(ctx context.Context, subType SubscriptionType, event interface{}) error {
	c.callbacksLock.RLock()
	callbacks := c.callbacks[subType]
	c.callbacksLock.RUnlock()

	if len(callbacks) == 0 {
		return nil
	}

	for i, callback := range callbacks {
		if err := callback(ctx, event); err != nil {
			c.log.WithFields(logrus.Fields{
				"subscription_type": subType,
				"callback_index":    i,
				"error":             err.Error(),
			}).Error("Callback error")
		}
	}

	return nil
}

// getRegisteredSubscriptionTypes returns all subscription types that have callbacks registered.
func (c *Client) getRegisteredSubscriptionTypes() []SubscriptionType {
	c.callbacksLock.RLock()
	defer c.callbacksLock.RUnlock()

	subTypes := make([]SubscriptionType, 0)

	for subType, callbacks := range c.callbacks {
		if len(callbacks) > 0 {
			subTypes = append(subTypes, subType)
		}
	}

	return subTypes
}
