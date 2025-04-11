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
	// Address is the address of the execution client.
	Address string
	// Headers is a map of headers to send to the execution client.
	Headers map[string]string
	// PollingInterval is the interval to poll for new transactions when using HTTP(S) endpoints.
	PollingInterval time.Duration
}
