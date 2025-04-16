package execution

import (
	"context"
	"time"
)

// SubscriptionType represents a type of subscription to the execution client.
type SubscriptionType string

const (
	// SubNewPendingTransactions subscribes to new pending transactions.
	SubNewPendingTransactions SubscriptionType = "newPendingTransactions"
	// RPCMethodTxpoolContent is the RPC method for getting the content of the transaction pool.
	RPCMethodTxpoolContent = "txpool_content"
	// RPCMethodTxpoolInspect is the RPC method for inspecting the transaction pool.
	RPCMethodPendingTransactions = "eth_pendingTransactions"
)

// EventCallback is a generic callback function for subscription events.
type EventCallback func(ctx context.Context, event interface{}) error

// TransactionCallback is a callback function for when a transaction is received.
type TransactionCallback func(ctx context.Context, tx string) error

// Config represents execution client configuration.
type Config struct {
	// WSAddress is the WebSocket address of the execution client for subscriptions.
	// This is required for subscription functionality.
	WSAddress string

	// RPCAddress is the RPC address of the execution client.
	// This is required for all functionality.
	RPCAddress string

	// Headers is a map of headers to send to the execution client.
	Headers map[string]string

	// FetchInterval is the interval to fetch data from the execution client.
	FetchInterval time.Duration
}

// MempoolWatcherConfig represents configuration for the mempool watcher.
type MempoolWatcherConfig struct {
	// FetchInterval is how often to fetch txpool_content.
	FetchInterval time.Duration
	// PruneDuration is how long to keep pending txs in memory before pruning.
	PruneDuration time.Duration
}

// DefaultMempoolWatcherConfig returns a default configuration for the mempool watcher.
// The mempool watcher supports a multi-pronged approach to tracking transactions:
// 1. WebSocket subscription for new transaction hashes (doesn't include transaction data)
// 2. Regular polling of txpool_content (includes full transaction data)
// 3. High-frequency polling of eth_pendingTransactions (includes transaction data)
//
// Transactions are stored in PendingTxRecord which can include the transaction data
// when available. This avoids having to fetch the same transaction multiple times.
//
// The watcher uses a dedicated transaction processor that:
// 1. First processes transactions that already have data attached
// 2. Then processes transactions without data by batching RPC calls to fetch the data
//
// This approach maximizes efficiency by:
//   - Avoiding duplicate processing of transactions
//   - Prioritizing transactions that already have data
//   - Minimizing RPC calls through batching
//   - Ensuring all data sources (WebSocket, txpool_content, eth_pendingTransactions)
//     contribute to a single unified transaction store
func DefaultMempoolWatcherConfig() *MempoolWatcherConfig {
	return &MempoolWatcherConfig{
		FetchInterval: 15 * time.Second,
		PruneDuration: 5 * time.Minute,
	}
}
