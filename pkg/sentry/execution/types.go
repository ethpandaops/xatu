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
func DefaultMempoolWatcherConfig() *MempoolWatcherConfig {
	return &MempoolWatcherConfig{
		FetchInterval: 15 * time.Second,
		PruneDuration: 5 * time.Minute,
	}
}
