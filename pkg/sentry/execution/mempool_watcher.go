package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/sirupsen/logrus"
	"github.com/sony/gobreaker/v2"
)

// MempoolWatcher captures and processes pending transactions from the Ethereum execution client.
// It uses three complementary methods to ensure comprehensive transaction coverage:
// 1. Websocket subscription to newPendingTransactions for real-time notification
// 2. Periodic polling of txpool_content for full transaction details
// 3. Periodic polling of eth_pendingTransactions as a fallback/supplementary source
//
// Transactions flow through the system as follows:
// - Detected via any of the three sources above
// - Added to pendingTxs map (temporary storage until processed)
// - Queued for processing via txQueue
// - Processed by worker goroutines
// - Marked as processed in txQueue.processed map to prevent reprocessing
// - Removed from pendingTxs map
//
// The pendingTxs map functions as a temporary workspace for in-flight processing,
// not as a permanent mirror of the mempool state.
type MempoolWatcher struct {
	client                  ClientProvider
	log                     logrus.FieldLogger
	config                  *Config
	pendingTxs              map[string]*PendingTxRecord
	pendingTxsMutex         sync.RWMutex
	wg                      sync.WaitGroup
	ctx                     context.Context //nolint:containedctx // This is a derived context from the parent context.
	cancel                  context.CancelFunc
	txQueue                 *txQueue
	subscriptionCancel      context.CancelFunc // for canceling active subscription
	processTxCallback       func(context.Context, *PendingTxRecord, json.RawMessage) error
	metrics                 *Metrics
	txsReceivedBySource     map[string]int64
	metricsMutex            sync.RWMutex
	txsByHashRpcBreaker     *gobreaker.CircuitBreaker[[]json.RawMessage]
	txPoolContentRpcBreaker *gobreaker.CircuitBreaker[json.RawMessage]
	pendingTxsBreaker       *gobreaker.CircuitBreaker[[]json.RawMessage]
}

// NewMempoolWatcher creates a new MempoolWatcher with configured circuit breakers
// for resilient RPC operations
func NewMempoolWatcher(
	client ClientProvider,
	log logrus.FieldLogger,
	config *Config,
	processTxCallback func(context.Context, *PendingTxRecord, json.RawMessage) error,
	metrics *Metrics,
) *MempoolWatcher {
	var (
		ctx    context.Context
		cancel context.CancelFunc
	)

	// Create a circuit breaker for batch RPC calls.
	txsByHashRpcBreaker := gobreaker.NewCircuitBreaker[[]json.RawMessage](gobreaker.Settings{
		Name:        "el-rpc-breaker",
		MaxRequests: 1,                                                              // Allow only 1 request when half-open
		Interval:    0,                                                              // Don't reset counts until state change
		Timeout:     time.Duration(config.CircuitBreakerResetTimeout) * time.Second, // Time to wait in open state before trying again
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip after configured consecutive failures
			return counts.ConsecutiveFailures > uint32(config.CircuitBreakerFailureThreshold) //nolint:gosec // G115: intentional conversionwg.Add(1)
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.WithFields(logrus.Fields{
				"from_state": from.String(),
				"to_state":   to.String(),
			}).Info("Circuit breaker state changed")

			// Update metrics when state changes
			metrics.SetCircuitBreakerState("el-rpc-breaker", to.String())
		},
	})

	// Create a circuit breaker for txpool_content RPC calls.
	txPoolContentRpcBreaker := gobreaker.NewCircuitBreaker[json.RawMessage](gobreaker.Settings{
		Name:        "el-single-rpc-breaker",
		MaxRequests: 1,                                                              // Allow only 1 request when half-open
		Interval:    0,                                                              // Don't reset counts until state change
		Timeout:     time.Duration(config.CircuitBreakerResetTimeout) * time.Second, // Time to wait in open state before trying again
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip after configured consecutive failures
			return counts.ConsecutiveFailures > uint32(config.CircuitBreakerFailureThreshold) //nolint:gosec // G115: intentional conversion
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.WithFields(logrus.Fields{
				"from_state": from.String(),
				"to_state":   to.String(),
			}).Info("Circuit breaker state changed")

			// Update metrics when state changes
			metrics.SetCircuitBreakerState("el-single-rpc-breaker", to.String())
		},
	})

	// Create a circuit breaker for pending transactions.
	pendingTxsBreaker := gobreaker.NewCircuitBreaker[[]json.RawMessage](gobreaker.Settings{
		Name:        "el-pending-tx-breaker",
		MaxRequests: 1,                                                              // Allow only 1 request when half-open
		Interval:    0,                                                              // Don't reset counts until state change
		Timeout:     time.Duration(config.CircuitBreakerResetTimeout) * time.Second, // Time to wait in open state before trying again
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip after configured consecutive failures
			return counts.ConsecutiveFailures > uint32(config.CircuitBreakerFailureThreshold) //nolint:gosec // G115: intentional conversion
		},
		OnStateChange: func(name string, from gobreaker.State, to gobreaker.State) {
			log.WithFields(logrus.Fields{
				"from_state": from.String(),
				"to_state":   to.String(),
			}).Info("Circuit breaker state changed")

			// Update metrics when state changes
			metrics.SetCircuitBreakerState("el-pending-tx-breaker", to.String())
		},
	})

	return &MempoolWatcher{
		client:                  client,
		config:                  config,
		log:                     log.WithField("component", "execution/mempool_watcher"),
		pendingTxs:              make(map[string]*PendingTxRecord),
		pendingTxsMutex:         sync.RWMutex{},
		wg:                      sync.WaitGroup{},
		ctx:                     ctx,
		cancel:                  cancel,
		txQueue:                 newTxQueue(metrics, config.QueueSize, time.Duration(config.PruneDuration)*time.Second),
		processTxCallback:       processTxCallback,
		metrics:                 metrics,
		txsReceivedBySource:     make(map[string]int64),
		metricsMutex:            sync.RWMutex{},
		txsByHashRpcBreaker:     txsByHashRpcBreaker,
		txPoolContentRpcBreaker: txPoolContentRpcBreaker,
		pendingTxsBreaker:       pendingTxsBreaker,
	}
}

// Start initializes the watcher's context and launches all background goroutines
// for transaction discovery and processing.
func (w *MempoolWatcher) Start(ctx context.Context) error {
	// Validate configuration values before starting
	if w.config.ProcessingInterval <= 0 {
		return fmt.Errorf("processingInterval must be positive, got %d", w.config.ProcessingInterval)
	}

	if w.config.PruneDuration <= 0 {
		return fmt.Errorf("pruneDuration must be positive, got %d", w.config.PruneDuration)
	}

	if w.config.FetchInterval <= 0 {
		return fmt.Errorf("fetchInterval must be positive, got %d", w.config.FetchInterval)
	}

	if w.config.ProcessorWorkerCount <= 0 {
		return fmt.Errorf("processorWorkerCount must be positive, got %d", w.config.ProcessorWorkerCount)
	}

	if w.config.RpcBatchSize <= 0 {
		return fmt.Errorf("rpcBatchSize must be positive, got %d", w.config.RpcBatchSize)
	}

	if w.config.MaxConcurrency <= 0 {
		return fmt.Errorf("maxConcurrency must be positive, got %d", w.config.MaxConcurrency)
	}

	// Create a derived context that can be canceled independently of the parent context.
	w.ctx, w.cancel = context.WithCancel(ctx)

	// Start WebSocket subscription for new pending transactions (if enabled).
	if w.config.WebsocketEnabled {
		if err := w.startNewPendingTxSubscription(); err != nil {
			return fmt.Errorf("failed to start websocket subscription: %w", err)
		}
	}

	// Start periodic fetching of txpool content.
	if w.config.TxPoolContentEnabled {
		w.startPeriodicFetcher()
	}

	// Start periodic fetching of eth_pendingTransactions.
	if w.config.EthPendingTxsEnabled {
		w.startPendingTransactionsFetcher()
	}

	// Start periodic summary logging.
	w.startSummaryLogger()

	// Start periodic pruning of old pending/processed txs.
	w.startPruner()

	// Start transaction processor - worker pool to consume from the queue.
	w.startTransactionProcessor()

	w.log.Info("Started mempool watcher")

	return nil
}

// Stop cleanly shuts down the watcher, canceling the WebSocket subscription
// and waiting for all goroutines to complete.
func (w *MempoolWatcher) Stop() {
	w.log.Info("Stopping mempool watcher")

	// Unsubscribe from WebSocket subscription if active.
	if w.subscriptionCancel != nil {
		w.subscriptionCancel()
	}

	w.cancel()
	w.wg.Wait()
}

// startNewPendingTxSubscription creates a persistent WebSocket subscription with automatic
// reconnection using exponential backoff. This ensures we receive real-time notifications
// of new transactions as they enter the mempool.
func (w *MempoolWatcher) startNewPendingTxSubscription() error {

	w.wg.Go(func() {

		ctx, cancel := context.WithCancel(w.ctx)
		defer cancel() // This hits when the goroutine exits.

		// Create exponential backoff, if we loose the connection, we should retry.
		bo := backoff.NewExponentialBackOff()
		bo.MaxInterval = 1 * time.Minute

		operation := func() (string, error) {
			// Have we've been told to stop?
			select {
			case <-ctx.Done():
				return "", backoff.Permanent(fmt.Errorf("context canceled"))
			default:
			}

			// Attempt socket subscription, on failure, we'll retry.
			if err := w.subscribeToNewPendingTransactions(); err != nil {
				return "", err
			}

			// Create a channel to signal when the subscription ends.
			done := make(chan error, 1)

			// Wait in a goroutine for the subscription to end.
			go func() {
				// The subscription will end when w.subscriptionCancel is called
				// or when the parent context is canceled.
				<-ctx.Done()
				done <- fmt.Errorf("context canceled")
			}()

			// Wait for signal that subscription has ended.
			err := <-done

			bo.Reset()

			return "", err
		}

		// Configure subscription retry options.
		retryOpts := []backoff.RetryOption{
			backoff.WithBackOff(bo),
			backoff.WithNotify(func(err error, timer time.Duration) {
				w.log.WithError(err).WithFields(logrus.Fields{
					"next_attempt": timer,
					"topic":        SubNewPendingTransactions,
				}).Error("websocket subscription failed, will retry")
			}),
		}

		// Retry with backoff until we get a permanent error.
		if _, err := backoff.Retry(ctx, operation, retryOpts...); err != nil {
			w.log.WithError(err).Error("websocket subscription permanently failed")
		}
	})

	return nil
}

// subscribeToNewPendingTransactions establishes the actual WebSocket subscription
// and sets up a goroutine to process transaction notifications.
func (w *MempoolWatcher) subscribeToNewPendingTransactions() error {
	// Always set the connected status to false at the beginning.
	w.metrics.SetWebsocketConnected(false)

	// Create a context for this subscription that can be canceled
	subCtx, cancel := context.WithCancel(w.ctx)
	w.subscriptionCancel = cancel

	// Subscribe to new pending transactions
	txChan, errChan, err := w.client.SubscribeToNewPendingTxs(subCtx)
	if err != nil {
		w.subscriptionCancel = nil

		cancel() // Clean up if subscription fails

		w.log.WithError(err).Error("Failed to subscribe to newPendingTransactions")

		return err
	}

	// We've got a connection, :tada:
	w.metrics.SetWebsocketConnected(true)

	w.log.WithField("topic", SubNewPendingTransactions).Debug("Subscribed to newPendingTransactions")

	// Start goroutine to process incoming transactions.
	go func() {
		defer cancel() // Ensure context is canceled when goroutine exits

		for {
			select {
			case <-subCtx.Done():
				w.log.Debug("subscription context canceled, stopping transaction processing")
				w.metrics.SetWebsocketConnected(false)

				return
			case txHash := <-txChan:
				w.addPendingTransaction(txHash, nil, fmt.Sprintf("ws_%s", SubNewPendingTransactions))
			case err := <-errChan:
				w.log.WithError(err).Error("subscription error")
				w.metrics.SetWebsocketConnected(false)

				return
			}
		}
	}()

	return nil
}

// startPeriodicFetcher launches a goroutine that periodically fetches the full
// txpool content to get comprehensive transaction details.
func (w *MempoolWatcher) startPeriodicFetcher() {

	w.wg.Go(func() {

		ticker := time.NewTicker(time.Duration(w.config.FetchInterval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				if err := w.fetchAndProcessTxPool(w.ctx); err != nil {
					w.log.WithError(err).Error("Failed to fetch and process txpool content")
				}
			}
		}
	})
}

// fetchAndProcessTxPool retrieves and processes all transactions in the node's txpool.
// This provides complete transaction data and helps identify transactions that may have
// been missed by the WebSocket subscription. It also serves to identify transactions
// that have been dropped from the mempool.
func (w *MempoolWatcher) fetchAndProcessTxPool(ctx context.Context) error {
	startTime := time.Now()

	// Make RPC call to get txpool content. Wrap it in a circuit breaker.
	// The volume of calls we're making to source these transactions can
	// overwhelm the EL if we're not careful.
	var rawResponse json.RawMessage

	rawResponse, err := w.txPoolContentRpcBreaker.Execute(func() (json.RawMessage, error) {
		return w.client.GetTxpoolContent(ctx)
	})
	if err != nil {
		return err
	}

	// Parse the raw response into the structured map.
	var response map[string]any
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return fmt.Errorf("failed to unmarshal txpool_content response: %w", err)
	}

	var (
		fetchTime        = time.Since(startTime)
		processStartTime = time.Now()
	)

	w.metrics.ObserveRPCRequestDuration(string(RPCMethodTxpoolContent), fetchTime.Seconds())

	// Parse the response and add transactions to the pendingTxs map.
	totalTxCount := 0
	newWithDataCount := 0
	updatedWithDataCount := 0
	duplicateCount := 0

	// Get a set of all transaction hashes in the txpool.
	allTxHashes := make(map[string]bool)

	// Process pending and queued transactions.
	for _, section := range []string{"pending", "queued"} {
		sectionData, ok := response[section].(map[string]any)
		if !ok {
			continue
		}

		for _, accountData := range sectionData {
			accountTxs, ok := accountData.(map[string]any)
			if !ok {
				continue
			}

			// Iterate through each transaction.
			for _, txData := range accountTxs {
				totalTxCount++

				// Extract transaction hash from the data.
				txDataMap, ok := txData.(map[string]any)
				if !ok {
					continue
				}

				txHash, ok := txDataMap["hash"].(string)
				if !ok {
					continue
				}

				// Add to the set of known txs in the txpool.
				allTxHashes[txHash] = true

				// Convert the txData into a JSON byte array for later processing.
				txDataBytes, err := json.Marshal(txData)
				if err != nil {
					w.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to marshal transaction data")

					continue
				}

				// Check if we've already processed this transaction.
				if w.txQueue.isProcessed(txHash) {
					duplicateCount++

					continue
				}

				// If not, we're good to go! Add it for processing.
				w.addPendingTransaction(txHash, txDataBytes, fmt.Sprintf("rpc_%s", RPCMethodTxpoolContent))
			}
		}
	}

	// Now check potentially dropped transactions against what we found in txpool_content.
	// Why? If we've got tx's in our pending map that are marked for pruning, and not
	// found in the txpool content, they've been dropped (think, cancelled tx's). So, given
	// we have the full txpool content, we can see if they're still in the txpool, and if not,
	// remove them from our pending map and mark them as processed.
	var (
		removedCount int
		txsToRemove  = make([]string, 0)
	)

	w.pendingTxsMutex.RLock()

	for hash, record := range w.pendingTxs {
		// If transaction is marked for pruning and not found in txpool content,
		// add it to the list of txs to remove.
		if record.MarkedForPruning && !allTxHashes[hash] {
			txsToRemove = append(txsToRemove, hash)
		}
	}

	w.pendingTxsMutex.RUnlock()

	// Now remove the collected transactions (if any).
	if len(txsToRemove) > 0 {
		removedCount = len(txsToRemove)

		// Mark as processed in the queue.
		for _, hash := range txsToRemove {
			w.txQueue.markProcessed(hash)
		}

		// Then remove from pending map.
		w.pendingTxsMutex.Lock()
		for _, hash := range txsToRemove {
			delete(w.pendingTxs, hash)
		}
		w.pendingTxsMutex.Unlock()

		w.txQueue.recordProcessingOutcome("pruned", removedCount)
	}

	var (
		processTime = time.Since(processStartTime)
		totalTime   = time.Since(startTime)
	)

	// Record batch processing duration in metrics
	w.metrics.ObserveBatchProcessingDuration(processTime.Seconds())

	w.pendingTxsMutex.RLock()
	pendingCount := len(w.pendingTxs)
	w.pendingTxsMutex.RUnlock()

	w.log.WithFields(logrus.Fields{
		"pending_txs":        pendingCount,
		"txpool_txs":         totalTxCount,
		"new_with_data":      newWithDataCount,
		"updated_with_data":  updatedWithDataCount,
		"duplicate_txs":      duplicateCount,
		"removed_pruned_txs": removedCount,
		"fetch_time":         fetchTime.String(),
		"process_time":       processTime.String(),
		"total_time":         totalTime.String(),
	}).Debug("[txpool_content] Processed mempool transactions")

	return nil
}

// startPendingTransactionsFetcher launches a goroutine that periodically fetches
// eth_pendingTransactions as a supplementary source of transactions.
// This method runs more frequently than txpool_content and has jitter added to
// avoid creating predictable load patterns on the node.
func (w *MempoolWatcher) startPendingTransactionsFetcher() {

	w.wg.Go(func() {

		// Stagger this slightly offset from txpool_content fetch to avoid overwhelming the EL.
		// Use 1/3 of the fetchInterval to have this run 3x as frequently as txpool_content
		fetchInterval := time.Duration(w.config.FetchInterval) * time.Second / 3

		// Apply jitter to avoid thundering herd problem and predictable load spikes
		// This adds random variance between 10% and 90% of the fetchInterval.
		jitterDuration := time.Duration(float64(fetchInterval) * (0.1 + rand.Float64()*0.8)) //nolint:gosec // G404: non-cryptographic randomness is sufficient for jitter
		time.Sleep(jitterDuration)

		ticker := time.NewTicker(fetchInterval)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				if err := w.fetchAndProcessPendingTransactions(w.ctx); err != nil {
					w.log.WithError(err).Error("Failed to fetch and process pending transactions")
				}
			}
		}
	})
}

// fetchAndProcessPendingTransactions fetches eth_pendingTransactions to discover new transactions.
func (w *MempoolWatcher) fetchAndProcessPendingTransactions(ctx context.Context) error {
	startTime := time.Now()

	// Fetch pending transactions directly using the pendingTxsBreaker
	transactions, err := w.pendingTxsBreaker.Execute(func() ([]json.RawMessage, error) {
		return w.client.GetPendingTransactions(ctx)
	})

	if err != nil {
		return err
	}

	fetchTime := time.Since(startTime)
	w.metrics.ObserveRPCRequestDuration(string(RPCMethodPendingTransactions), fetchTime.Seconds())

	// No transactions found, we're done.
	if len(transactions) == 0 {
		return nil
	}

	var (
		processStartTime = time.Now()
		processedCount   = 0
		duplicateCount   = 0
		newTxCount       = 0
		allTxHashes      = make(map[string]bool)
	)

	// Process each transaction.
	for _, txData := range transactions {
		// Extract hash from transaction data.
		var tx struct {
			Hash string `json:"hash"`
		}

		if err := json.Unmarshal(txData, &tx); err != nil {
			w.log.WithError(err).Debug("Failed to parse transaction data from eth_pendingTransactions")

			continue
		}

		txHash := tx.Hash

		// Add to set of known transactions.
		allTxHashes[txHash] = true

		// Check if we've already processed this transaction.
		if w.txQueue.isProcessed(txHash) {
			duplicateCount++

			continue
		}

		processedCount++
		newTxCount++

		// We're good, add it to be processed.
		w.addPendingTransaction(txHash, txData, fmt.Sprintf("rpc_%s", RPCMethodPendingTransactions))
	}

	// Now we can check for any transactions that were marked for pruning but are still
	// in the txpool. If so, we can clear the flag and let it continue to be processed.
	txsToClear := make([]string, 0)

	w.pendingTxsMutex.RLock()

	for hash, record := range w.pendingTxs {
		if record.MarkedForPruning && allTxHashes[hash] {
			txsToClear = append(txsToClear, hash)
		}
	}

	w.pendingTxsMutex.RUnlock()

	if len(txsToClear) > 0 {
		for _, hash := range txsToClear {
			w.log.WithField("tx_hash", hash).Debug("Transaction previously marked for pruning was found in eth_pendingTransactions, clearing flag")
		}

		w.pendingTxsMutex.Lock()
		for _, hash := range txsToClear {
			if record, exists := w.pendingTxs[hash]; exists && record.MarkedForPruning {
				record.MarkedForPruning = false
			}
		}
		w.pendingTxsMutex.Unlock()
	}

	var (
		processTime = time.Since(processStartTime)
		totalTime   = time.Since(startTime)
	)

	w.metrics.ObserveBatchProcessingDuration(processTime.Seconds())

	w.log.WithFields(logrus.Fields{
		"total_txs":     len(transactions),
		"processed_txs": processedCount,
		"duplicate_txs": duplicateCount,
		"new_txs":       newTxCount,
		"fetch_time":    fetchTime.String(),
		"process_time":  processTime.String(),
		"total_time":    totalTime.String(),
	}).Debug("[eth_pendingTransactions] Processed mempool transactions")

	return nil
}

// startTransactionProcessor launches worker goroutines to process transactions
// from the queue. It includes two types of workers:
// 1. Direct processors that handle transactions with data already available.
// 2. A batch processor that collects and fetches data for transactions without details.
func (w *MempoolWatcher) startTransactionProcessor() {
	// Start worker goroutines to process transactions with data directly
	// We allocate half of the configured workers to direct processing (transactions with data),
	// while the other half is implicitly reserved for batch processing transactions that need data fetched.
	for i := 0; i < w.config.ProcessorWorkerCount/2; i++ {
		w.wg.Add(1)

		go func(workerID int) {
			defer w.wg.Done()

			for {
				select {
				case <-w.ctx.Done():
					return
				case item := <-w.txQueue.items:
					w.txQueue.updateQueueMetrics()

					// First, process transactions that already have data. Remembering, tx's received via
					// socket don't come hydrated, so we need to fetch their in a separate process.
					if item.txData != nil {
						if err := w.processTxCallback(w.ctx, item.record, item.txData); err != nil {
							w.log.WithError(err).WithField("tx_hash", item.record.Hash).
								WithField("worker", workerID).
								Error("Failed to process transaction with data")

							w.txQueue.recordProcessingOutcome("error", 1)
						} else {
							w.txQueue.recordProcessingOutcome("success", 1)
						}

						// Mark as processed and remove from pending map.
						w.txQueue.markProcessed(item.record.Hash)

						w.pendingTxsMutex.Lock()
						delete(w.pendingTxs, item.record.Hash)
						w.pendingTxsMutex.Unlock()
					} else {
						// If it doesn't have any data yet, put it back in the pending map to be picked
						// up by the batch processor which will fetch the data for it.
						w.pendingTxsMutex.Lock()
						if _, exists := w.pendingTxs[item.record.Hash]; !exists {
							w.pendingTxs[item.record.Hash] = item.record
						}
						w.pendingTxsMutex.Unlock()
					}
				}
			}
		}(i)
	}

	// Start a separate goroutine for batch processing of transactions without data.

	w.wg.Go(func() {

		ticker := time.NewTicker(time.Duration(w.config.ProcessingInterval) * time.Millisecond)
		defer ticker.Stop()

		// Use a mutex to prevent overlapping batch processing.
		var batchMutex sync.Mutex

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				// A quick check to prevent overlapping of the batch processing.
				if batchMutex.TryLock() {
					go func() {
						defer batchMutex.Unlock()
						w.processPendingTransactionsBatch()
					}()
				}
			}
		}
	})

	// Start a goroutine to periodically prune the processed transactions map.

	w.wg.Go(func() {

		ticker := time.NewTicker(time.Duration(w.config.PruneDuration) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				prunedCount := w.txQueue.prune()
				if prunedCount > 0 {
					w.log.WithField("pruned_count", prunedCount).Debug("Pruned processed transactions cache")
				}
			}
		}
	})
}

// processPendingTransactionsBatch collects transactions without full data and
// fetches their details in optimized batches. It uses circuit breaker patterns
// to handle RPC failures gracefully and adjusts concurrency based on circuit state.
func (w *MempoolWatcher) processPendingTransactionsBatch() {
	batchStartTime := time.Now()

	// Get all pending transactions without data.
	var txsWithoutData map[string]*PendingTxRecord

	w.pendingTxsMutex.RLock()

	// Create a copy of the transactions we want to process.
	txsWithoutData = make(map[string]*PendingTxRecord)

	for hash, record := range w.pendingTxs {
		if record.TxData == nil && !w.txQueue.isProcessed(hash) {
			txsWithoutData[hash] = record
		}
	}

	w.pendingTxsMutex.RUnlock()

	// If no transactions to process, return early.
	if len(txsWithoutData) == 0 {
		return
	}

	// Create batches of transaction hashes to fetch.
	var (
		txBatches    [][]string
		batchSize    = w.config.RpcBatchSize
		currentBatch = make([]string, 0, batchSize)
	)

	for hash := range txsWithoutData {
		currentBatch = append(currentBatch, hash)

		if len(currentBatch) >= batchSize {
			txBatches = append(txBatches, currentBatch)
			currentBatch = make([]string, 0, batchSize)
		}
	}

	// Add any remaining transactions.
	if len(currentBatch) > 0 {
		txBatches = append(txBatches, currentBatch)
	}

	// Adjust concurrency based on circuit breaker state.
	var (
		maxConcurrency = w.config.MaxConcurrency
		circuitState   = w.txsByHashRpcBreaker.State()
	)

	// Reduce concurrency when the circuit breaker is in half-open state. When the circuit breaker moves
	// to the half-open state, it means that it's trying to re-establish the connection, we don't want to
	// go back into the EL with a full-head of steam. Ease into it.
	switch circuitState {
	case gobreaker.StateHalfOpen:
		maxConcurrency = 1

		w.log.Debug("circuit breaker in half-open state, reducing batch concurrency to 1")
	case gobreaker.StateOpen:
		w.log.Debug("circuit breaker in open state, skipping batch processing")

		return
	default:
	}

	var (
		sem = make(chan struct{}, maxConcurrency)
		wg  sync.WaitGroup
	)

	// Process each batch.
	for batchIndex, batch := range txBatches {
		sem <- struct{}{}

		wg.Add(1)

		go func(batchNum int, hashes []string) {
			batchRequestTime := time.Now()

			defer func() {
				<-sem
				wg.Done()
			}()

			// Execute batch request.
			results, err := w.txsByHashRpcBreaker.Execute(func() ([]json.RawMessage, error) {
				return w.client.BatchGetTransactionsByHash(w.ctx, hashes)
			})

			// Record RPC request duration in metrics.
			requestDuration := time.Since(batchRequestTime)
			w.metrics.ObserveRPCRequestDuration(RPCMethodGetTransactionByHash, requestDuration.Seconds())

			if err != nil {
				w.log.WithError(err).WithField("batch", batchNum).Error("Batch transaction fetch failed")

				w.txQueue.recordProcessingOutcome("batch_error", len(hashes))

				return
			}

			// Process results.
			for i, hash := range hashes {
				if i >= len(results) || len(results[i]) == 0 {
					// Update outcome metrics
					w.txQueue.recordProcessingOutcome("skipped", 1)

					continue
				}

				// Check if result is null. How would we end up with a null result you may ask?
				// If we request a transaction that is not in the mempool, the EL's will return "null".
				// This can happen if the transaction is not in the mempool anymore, say perhaps the
				// tx was cancelled, replaced or was added to a block.
				//
				// If we receive a null result, we need to mark the transaction for pruning from our
				// local pending map. It won't be removed instantly. Next time txpool_content is requested,
				// we will double-check its not in the mempool and then remove it.
				if string(results[i]) == "null" {
					// Record the processing outcome for analytics.
					w.txQueue.recordProcessingOutcome("null", 1)

					w.pendingTxsMutex.Lock()
					if record, exists := w.pendingTxs[hash]; exists && !record.MarkedForPruning {
						record.MarkedForPruning = true
					}
					w.pendingTxsMutex.Unlock()

					w.txQueue.markProcessed(hash)

					w.pendingTxsMutex.Lock()
					delete(w.pendingTxs, hash)
					w.pendingTxsMutex.Unlock()

					continue
				}

				// Get the record.
				record, exists := txsWithoutData[hash]
				if !exists {
					w.txQueue.recordProcessingOutcome("skipped", 1)

					continue
				}

				// Update the record with the data.
				record.TxData = results[i]

				// Process the transaction.
				if err := w.processTxCallback(w.ctx, record, results[i]); err != nil {
					w.log.WithError(err).WithField("tx_hash", hash).Error("Failed to process transaction")

					w.txQueue.recordProcessingOutcome("error", 1)
				} else {
					w.txQueue.recordProcessingOutcome("success", 1)
				}

				// Mark as processed and remove from pending.
				w.txQueue.markProcessed(hash)

				w.pendingTxsMutex.Lock()
				delete(w.pendingTxs, hash)
				w.pendingTxsMutex.Unlock()
			}
		}(batchIndex, batch)
	}

	// Wait for all batches to complete.
	wg.Wait()
	close(sem)

	batchProcessingTime := time.Since(batchStartTime)
	w.metrics.ObserveBatchProcessingDuration(batchProcessingTime.Seconds())
}

// addPendingTransaction adds a transaction to the pending map and processing queue.
// It handles several cases:
// - New transactions: Added to pendingTxs and queued for processing.
// - Existing transactions with new data: Updated with data and requeued.
// - Already processed transactions: Skipped entirely.
// - Transactions marked for pruning but seen again: Mark cleared.
func (w *MempoolWatcher) addPendingTransaction(txHash string, txData json.RawMessage, source string) {
	// Normalize the hash (ensure it has 0x prefix).
	if len(txHash) >= 2 && txHash[:2] != "0x" {
		txHash = "0x" + txHash
	}

	// Check if already processed using our queue's processed map.
	if w.txQueue.isProcessed(txHash) {
		return
	}

	// Check if it's already in our pending map.
	w.pendingTxsMutex.Lock()
	record, exists := w.pendingTxs[txHash]

	if exists {
		// Already in pending map, but we might need to update the TxData if we didn't have it before.
		// Tx's added to our map via the socket don't come with the TxData, so we need to fetch it.
		if record.TxData == nil && txData != nil {
			record.TxData = txData
		}

		// If this transaction was previously marked for pruning, receiving it again from any
		// source means it's still active on the network. So we clear the pruning mark.
		if record.MarkedForPruning {
			record.MarkedForPruning = false

			w.log.WithFields(logrus.Fields{
				"tx_hash": txHash,
				"source":  source,
			}).Debug("Cleared pruning mark for transaction still active in the network")
		}
	} else {
		// It's a new transaction to us, create a record.
		record = &PendingTxRecord{
			Hash:             txHash,
			FirstSeen:        time.Now(),
			Attempts:         0,
			TxData:           txData,
			Source:           source,
			MarkedForPruning: false,
		}

		// Add to pending map + metrics.
		w.pendingTxs[txHash] = record
		w.metricsMutex.Lock()
		w.txsReceivedBySource[source]++
		w.metricsMutex.Unlock()
		w.txQueue.markTxReceived(source)

		// Update the pending count.
		pendingCount := len(w.pendingTxs)
		w.txQueue.setTxPending(pendingCount)
	}
	w.pendingTxsMutex.Unlock()

	// Add to processing queue (only if not already in the queue).
	if !exists || (exists && record.TxData == nil && txData != nil) {
		// Clone the record to avoid any potential race conditions.
		recordCopy := &PendingTxRecord{
			Hash:             record.Hash,
			FirstSeen:        record.FirstSeen,
			Attempts:         record.Attempts,
			TxData:           record.TxData,
			Source:           record.Source,
			MarkedForPruning: record.MarkedForPruning,
		}

		// Try to add to queue, ignore if queue is full or transaction already processed.
		// Ignoring bool response here, as tracking of rejected txs is handled inside txQueue.add.
		_ = w.txQueue.add(recordCopy, txData)
	}
}

// startPruner launches a goroutine that periodically removes old pending transactions
// based on age. This is a fallback cleanup mechanism for transactions that weren't
// explicitly confirmed or dropped.
func (w *MempoolWatcher) startPruner() {

	w.wg.Go(func() {

		//  // Run pruner at half the prune duration.
		ticker := time.NewTicker(time.Duration(w.config.PruneDuration) * time.Second / 2)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.prunePendingTxs()
			}
		}
	})
}

// prunePendingTxs removes transactions that have been in the pending map longer than
// the configured pruning duration. This is a safety net for cleaning up transactions
// that were neither confirmed nor explicitly removed through other mechanisms.
//
// There are two pruning mechanisms in the system:
//  1. MarkedForPruning flag-based: When a transaction is found to be null or missing
//     from the mempool, it's marked and then removed when verified missing.
//  2. Time-based: This method, which removes transactions based solely on age.
func (w *MempoolWatcher) prunePendingTxs() {
	w.pendingTxsMutex.Lock()
	defer w.pendingTxsMutex.Unlock()

	now := time.Now()
	prunedCount := 0

	for hash, record := range w.pendingTxs {
		if now.Sub(record.FirstSeen) <= time.Duration(w.config.PruneDuration)*time.Second {
			continue
		}

		delete(w.pendingTxs, hash)

		prunedCount++

		w.txQueue.recordProcessingOutcome("expired", 1)

		pendingCount := len(w.pendingTxs)
		w.txQueue.setTxPending(pendingCount)
	}

	if prunedCount > 0 {
		w.log.WithField("pruned_count", prunedCount).Info("Pruned old pending transactions")
	}
}

// startSummaryLogger launches a goroutine that periodically logs operational metrics
// about the mempool watcher state for monitoring purposes.
func (w *MempoolWatcher) startSummaryLogger() {

	w.wg.Go(func() {

		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.logMetrics()
			}
		}
	})
}

// logMetrics records and logs key operational metrics about the mempool watcher state.
// This includes the number of pending transactions, queue size, circuit breaker state,
// and transactions marked for pruning.
func (w *MempoolWatcher) logMetrics() {
	// Get current state metrics from the pendingTxs map.
	w.pendingTxsMutex.RLock()
	pendingCount := len(w.pendingTxs)
	markedForPruningCount := 0

	// Count transactions marked for pruning.
	for _, record := range w.pendingTxs {
		if record.MarkedForPruning {
			markedForPruningCount++
		}
	}
	w.pendingTxsMutex.RUnlock()

	w.log.WithFields(logrus.Fields{
		"pending_count":      pendingCount,
		"marked_for_pruning": markedForPruningCount,
		"queue_size":         len(w.txQueue.items),
		"circuit_state":      w.txsByHashRpcBreaker.State().String(),
	}).Info("Mempool watcher stats")
}
