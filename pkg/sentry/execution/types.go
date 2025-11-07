package execution

import (
	"context"
	"encoding/json"
	"time"
)

// SubscriptionType represents a type of subscription to the execution client.
type SubscriptionType string

const (
	// SubNewPendingTransactions subscribes to new pending transactions.
	SubNewPendingTransactions SubscriptionType = "newPendingTransactions"
	// RPCMethodTxpoolContent is the RPC method for getting the content of the transaction pool.
	RPCMethodTxpoolContent = "txpool_content"
	// RPCMethodPendingTransactions is the RPC method for getting pending transactions.
	RPCMethodPendingTransactions = "eth_pendingTransactions"
	// RPCMethodGetTransactionByHash is the RPC method for getting a transaction by its hash.
	RPCMethodGetTransactionByHash = "eth_getTransactionByHash"
	// RPCMethodDebugStateSize is the RPC method for getting the execution layer state size.
	RPCMethodDebugStateSize = "debug_stateSize"
)

// Config defines configuration for connecting to an execution client.
type Config struct {
	// Enabled is whether the execution client is enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// EthPendingTxsEnabled is whether we suppliment fetching tx's using eth_pendingTransactions periodically.
	EthPendingTxsEnabled bool `yaml:"ethPendingTxsEnabled" default:"false"`
	// TxPoolContentEnabled is whether we suppliment fetching tx's using txpool_content periodically.
	TxPoolContentEnabled bool `yaml:"txPoolContentEnabled" default:"true"`
	// WebsocketEnabled is whether the websocket is enabled.
	WebsocketEnabled bool `yaml:"websocketEnabled" default:"false"`
	// WSAddress is the WebSocket address of the execution client for subscriptions.
	WSAddress string `yaml:"wsAddress"`
	// RPCAddress is the RPC address of the execution client for txpool_content calls.
	RPCAddress string `yaml:"rpcAddress"`
	// Headers is a map of headers to send to the execution client.
	Headers map[string]string `yaml:"headers"`
	// FetchInterval is how often to fetch txpool_content (in seconds).
	FetchInterval int `yaml:"fetchInterval" default:"15"`
	// PruneDuration is how long to keep pending transactions in memory before pruning (in seconds).
	PruneDuration int `yaml:"pruneDuration" default:"300"`
	// ProcessorWorkerCount is the number of worker goroutines for processing transactions.
	ProcessorWorkerCount int `yaml:"processorWorkerCount" default:"50"`
	// RpcBatchSize is the number of transactions to include in a single RPC batch call.
	RpcBatchSize int `yaml:"rpcBatchSize" default:"40"`
	// QueueSize is the size of the transaction processing queue.
	QueueSize int `yaml:"queueSize" default:"5000"`
	// ProcessingInterval is the interval at which to process batches of transactions (in milliseconds).
	ProcessingInterval int `yaml:"processingInterval" default:"500"`
	// MaxConcurrency is the maximum number of concurrent batch RPC requests.
	MaxConcurrency int `yaml:"maxConcurrency" default:"5"`
	// CircuitBreakerFailureThreshold is the number of consecutive failures before opening the circuit breaker.
	CircuitBreakerFailureThreshold int `yaml:"circuitBreakerFailureThreshold" default:"5"`
	// CircuitBreakerResetTimeout is the time to wait before transitioning from open to half-open (in seconds).
	CircuitBreakerResetTimeout int `yaml:"circuitBreakerResetTimeout" default:"30"`
	// StateSize is the configuration for state size monitoring.
	StateSize *StateSizeConfig `yaml:"stateSize"`
}

// PendingTxRecord represents a transaction hash and when it was first seen.
type PendingTxRecord struct {
	Hash             string
	FirstSeen        time.Time
	Attempts         int
	TxData           json.RawMessage // Raw tx data (when available).
	Source           string          // Eg: "websocket", "txpool_content", or "eth_pendingTransactions".
	MarkedForPruning bool
}

// EventCallback is a generic callback function for subscription events.
type EventCallback func(ctx context.Context, event interface{}) error

// TransactionCallback is a callback function for when a transaction is received.
type TransactionCallback func(ctx context.Context, tx string) error

// StateSizeConfig defines configuration for state size monitoring.
type StateSizeConfig struct {
	// Enabled is whether state size monitoring is enabled.
	Enabled bool `yaml:"enabled" default:"false"`
	// TriggerMode determines how state size polling is triggered.
	// Valid values: "head" (on consensus head events), "block" (on execution block events), "interval" (periodic polling).
	TriggerMode string `yaml:"triggerMode" default:"head"`
	// IntervalSeconds is the polling interval in seconds (used when TriggerMode is "interval").
	IntervalSeconds int `yaml:"intervalSeconds" default:"12"`
}

// DebugStateSizeResponse represents the response from debug_stateSize RPC call.
type DebugStateSizeResponse struct {
	AccountBytes         string `json:"accountBytes"`
	AccountTrienodeBytes string `json:"accountTrienodeBytes"`
	AccountTrienodes     string `json:"accountTrienodes"`
	Accounts             string `json:"accounts"`
	BlockNumber          string `json:"blockNumber"`
	ContractCodeBytes    string `json:"contractCodeBytes"`
	ContractCodes        string `json:"contractCodes"`
	StateRoot            string `json:"stateRoot"`
	StorageBytes         string `json:"storageBytes"`
	StorageTrienodeBytes string `json:"storageTrienodeBytes"`
	StorageTrienodes     string `json:"storageTrienodes"`
	Storages             string `json:"storages"`
}
