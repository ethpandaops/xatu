package execution

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/sirupsen/logrus"
)

// PendingTxRecord represents a transaction hash and when it was first seen
type PendingTxRecord struct {
	Hash               string
	FirstSeen          time.Time
	Attempts           int
	TxData             json.RawMessage // Raw transaction data when available
	Source             string          // Source of the transaction: "websocket", "txpool_content", or "eth_pendingTransactions"
	PotentiallyDropped bool            // Flag indicating if eth_getTransactionByHash returned null for this tx
}

// MempoolWatcher watches for pending transactions using a multi-pronged approach:
// 1. Subscribe to newPendingTransactions via websocket to get notified of new txs
// 2. Periodically fetch txpool_content to get full transaction details
// 3. Periodically fetch eth_pendingTransactions for additional transactions
type MempoolWatcher struct {
	client          *Client
	log             logrus.FieldLogger
	pendingTxs      map[string]*PendingTxRecord // Maps tx hash to when it was first seen
	pendingTxsMutex sync.RWMutex
	wg              sync.WaitGroup
	ctx             context.Context
	cancel          context.CancelFunc

	// WebSocket subscription management
	wsSubscription *rpc.ClientSubscription

	// Configuration
	fetchInterval time.Duration // How often to fetch txpool_content
	pruneDuration time.Duration // How long to keep pending txs in memory before pruning

	// Transaction processing configuration
	processingInterval time.Duration // How often to process pending transactions
	batchSize          int           // Maximum number of transactions to process in one batch

	// Callback when a transaction is processed with full data
	processTxCallback func(context.Context, *PendingTxRecord, json.RawMessage) error

	// Metrics
	txsReceivedCount        int64 // Total transactions received from all sources
	txsProcessedCount       int64 // Successfully processed transactions
	txsExpiredCount         int64 // Expired transactions
	txsNullCount            int64 // Transactions that returned null from fetch
	txsWithDataCount        int64 // Transactions received with data
	txsDataFetchedCount     int64 // Transactions that needed data to be fetched
	txsProcessWithDataCount int64 // Transactions processed with data
	txsFailedCount          int64 // Transactions that failed processing
	metricsMutex            sync.RWMutex

	// Cache of processed pending transactions to avoid duplicates
	processedPendingTxs      map[string]time.Time
	processedPendingTxsMutex sync.RWMutex
}

// NewMempoolWatcher creates a new MempoolWatcher
func NewMempoolWatcher(
	client *Client,
	log logrus.FieldLogger,
	fetchInterval time.Duration,
	pruneDuration time.Duration,
	processTxCallback func(context.Context, *PendingTxRecord, json.RawMessage) error,
) *MempoolWatcher {
	// Create a context that will be initialized in Start() instead of here
	var ctx context.Context
	var cancel context.CancelFunc

	return &MempoolWatcher{
		client:                   client,
		log:                      log.WithField("component", "execution/mempool_watcher"),
		pendingTxs:               make(map[string]*PendingTxRecord),
		pendingTxsMutex:          sync.RWMutex{},
		wg:                       sync.WaitGroup{},
		ctx:                      ctx,    // Will be initialized in Start()
		cancel:                   cancel, // Will be initialized in Start()
		fetchInterval:            fetchInterval,
		pruneDuration:            pruneDuration,
		processingInterval:       5 * time.Second, // Process pending transactions every 5 seconds
		batchSize:                100,             // Process up to 100 transactions at once
		processTxCallback:        processTxCallback,
		txsReceivedCount:         0,
		txsProcessedCount:        0,
		txsExpiredCount:          0,
		txsNullCount:             0,
		txsWithDataCount:         0,
		txsDataFetchedCount:      0,
		txsProcessWithDataCount:  0,
		txsFailedCount:           0,
		metricsMutex:             sync.RWMutex{},
		processedPendingTxs:      make(map[string]time.Time),
		processedPendingTxsMutex: sync.RWMutex{},
	}
}

// Start starts the mempool watcher
func (w *MempoolWatcher) Start(ctx context.Context) error {
	// Create a derived context that can be canceled independently
	w.ctx, w.cancel = context.WithCancel(ctx)

	// Start WebSocket subscription for new pending transactions
	if err := w.startNewPendingTxSubscription(); err != nil {
		return fmt.Errorf("failed to start WebSocket subscription: %w", err)
	}

	// Start periodic fetching of txpool content
	w.startPeriodicFetcher()

	// Start periodic fetching of eth_pendingTransactions
	w.startPendingTransactionsFetcher()

	// Start periodic metrics logging
	w.startMetricsLogger()

	// Start periodic pruning of old pending txs
	w.startPruner()

	// Start transaction processor - the new dedicated worker
	w.startTransactionProcessor()

	w.log.Info("Started mempool watcher")

	return nil
}

// Stop stops the mempool watcher
func (w *MempoolWatcher) Stop() {
	w.log.Info("Stopping mempool watcher")

	// Unsubscribe from WebSocket subscription if active
	if w.wsSubscription != nil {
		w.wsSubscription.Unsubscribe()
	}

	w.cancel()
	w.wg.Wait()
}

// startNewPendingTxSubscription subscribes to newPendingTransactions via WebSocket
// with automatic reconnection using backoff
func (w *MempoolWatcher) startNewPendingTxSubscription() error {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		// Create a context that is canceled when the main context is canceled
		ctx, cancel := context.WithCancel(w.ctx)
		defer cancel() // Ensure the context gets canceled when this goroutine exits

		// Create exponential backoff
		bo := backoff.NewExponentialBackOff()
		bo.MaxInterval = 1 * time.Minute

		// Operation to retry - subscribe to newPendingTransactions
		operation := func() (string, error) {
			// Check if we should stop
			select {
			case <-ctx.Done():
				return "", backoff.Permanent(fmt.Errorf("context canceled"))
			default:
			}

			// Try to subscribe
			err := w.subscribeToNewPendingTransactions()
			if err != nil {
				w.log.WithError(err).Error("Failed to subscribe, will retry after backoff")

				return "", err // Will be retried
			}

			// Wait for subscription to fail or context to be canceled
			select {
			case <-ctx.Done():
				return "", backoff.Permanent(fmt.Errorf("context canceled"))
			case err := <-w.wsSubscription.Err():
				w.wsSubscription = nil

				// Reset backoff to start fresh on next attempt
				bo.Reset()

				return "", err // Will be retried
			}
		}

		// Configure retry options.
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
	}()

	return nil
}

// subscribeToNewPendingTransactions subscribes to newPendingTransactions via WebSocket
func (w *MempoolWatcher) subscribeToNewPendingTransactions() error {
	ch := make(chan string)

	wsClient := w.client.GetWebSocketClient()
	if wsClient == nil {
		return fmt.Errorf("websocket client not initialized")
	}

	// Use w.ctx instead of context.Background() to respect the watcher's lifecycle
	sub, err := wsClient.EthSubscribe(w.ctx, ch, string(SubNewPendingTransactions))
	if err != nil {
		w.log.WithError(err).Error("Failed to subscribe to newPendingTransactions")
		return err
	}

	w.wsSubscription = sub

	w.log.WithField("topic", SubNewPendingTransactions).Info("Subscribed to newPendingTransactions")

	// Start goroutine to process incoming transactions
	go func() {
		for {
			select {
			// Add a case to handle w.ctx.Done() explicitly
			case <-w.ctx.Done():
				w.log.Debug("context canceled, stopping transaction processing")
				return
			case txHash := <-ch:
				w.addPendingTransaction(txHash, nil, "websocket")
			case err := <-w.wsSubscription.Err():
				w.log.WithError(err).Error("subscription error")
				// The subscription will be recreated by the outer loop
				return
			}
		}
	}()

	return nil
}

// startPeriodicFetcher starts a goroutine that periodically fetches txpool_content
func (w *MempoolWatcher) startPeriodicFetcher() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(w.fetchInterval)
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
	}()
}

// fetchAndProcessTxPool fetches txpool_content to discover new transactions with data
func (w *MempoolWatcher) fetchAndProcessTxPool(ctx context.Context) error {
	startTime := time.Now()

	// Make RPC call to get txpool content
	var response map[string]interface{}
	if err := w.client.CallContext(ctx, &response, RPCMethodTxpoolContent); err != nil {
		return err
	}

	fetchTime := time.Since(startTime)
	processStartTime := time.Now()

	// Parse the response and add transactions to the pendingTxs map
	totalTxCount := 0
	newWithDataCount := 0
	updatedWithDataCount := 0
	duplicateCount := 0

	// Get a set of all transaction hashes in the txpool
	allTxHashes := make(map[string]bool)

	// Process pending and queued transactions
	for _, section := range []string{"pending", "queued"} {
		sectionData, ok := response[section].(map[string]interface{})
		if !ok {
			continue
		}

		for _, accountData := range sectionData {
			accountTxs, ok := accountData.(map[string]interface{})
			if !ok {
				continue
			}

			// Iterate through each transaction
			for _, txData := range accountTxs {
				totalTxCount++

				// Extract transaction hash from the data
				txDataMap, ok := txData.(map[string]interface{})
				if !ok {
					continue
				}

				txHash, ok := txDataMap["hash"].(string)
				if !ok {
					continue
				}

				// Add to the set of known txs in the txpool
				allTxHashes[txHash] = true

				// Convert the txData into a JSON byte array for later processing
				txDataBytes, err := json.Marshal(txData)
				if err != nil {
					w.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to marshal transaction data")
					continue
				}

				// Check if we've already processed this transaction
				w.processedPendingTxsMutex.RLock()
				_, alreadyProcessed := w.processedPendingTxs[txHash]
				w.processedPendingTxsMutex.RUnlock()

				if alreadyProcessed {
					duplicateCount++
					continue
				}

				// Before adding, check if it would be an update to track metrics properly
				w.pendingTxsMutex.RLock()
				existingTx, exists := w.pendingTxs[txHash]
				isUpdate := exists && existingTx.TxData == nil // It's an update if we had the tx but no data
				w.pendingTxsMutex.RUnlock()

				// Add to pendingTxs using the unified helper
				w.addPendingTransaction(txHash, txDataBytes, "txpool_content")

				// Track metrics specific to this source
				if isUpdate {
					updatedWithDataCount++
				} else if !exists {
					newWithDataCount++
				}
			}
		}
	}

	// Now check potentially dropped transactions against what we found in txpool_content
	var removedCount int

	// First, safely collect transactions that need to be removed
	txsToRemove := make([]string, 0)

	w.pendingTxsMutex.RLock()
	for hash, record := range w.pendingTxs {
		// If transaction is potentially dropped and not found in txpool content
		if record.PotentiallyDropped && !allTxHashes[hash] {
			txsToRemove = append(txsToRemove, hash)
		}
	}
	w.pendingTxsMutex.RUnlock()

	// Now remove the collected transactions (if any)
	if len(txsToRemove) > 0 {
		removedCount = len(txsToRemove)

		// First add to processed map
		w.processedPendingTxsMutex.Lock()
		for _, hash := range txsToRemove {
			w.processedPendingTxs[hash] = time.Now()
		}
		w.processedPendingTxsMutex.Unlock()

		// Then remove from pending map
		w.pendingTxsMutex.Lock()
		for _, hash := range txsToRemove {
			delete(w.pendingTxs, hash)
		}
		w.pendingTxsMutex.Unlock()
	}

	processTime := time.Since(processStartTime)
	totalTime := time.Since(startTime)

	w.pendingTxsMutex.RLock()
	pendingCount := len(w.pendingTxs)
	w.pendingTxsMutex.RUnlock()

	w.log.WithFields(logrus.Fields{
		"pending_txs":         pendingCount,
		"txpool_txs":          totalTxCount,
		"new_with_data":       newWithDataCount,
		"updated_with_data":   updatedWithDataCount,
		"duplicate_txs":       duplicateCount,
		"removed_dropped_txs": removedCount,
		"fetch_time":          fetchTime.String(),
		"process_time":        processTime.String(),
		"total_time":          totalTime.String(),
	}).Debug("Fetched and processed txpool content")

	return nil
}

// startPendingTransactionsFetcher starts a goroutine that periodically fetches pending transactions using eth_pendingTransactions
func (w *MempoolWatcher) startPendingTransactionsFetcher() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		// Stagger this slightly offset from txpool_content fetch to avoid overwhelming the node
		// Use 1/3 of the fetchInterval to have this run 3x as frequently as txpool_content
		fetchInterval := w.fetchInterval / 2
		// Initial delay to avoid collision with txpool_content
		time.Sleep(fetchInterval / 2)

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
	}()
}

// fetchAndProcessPendingTransactions fetches eth_pendingTransactions to discover new transactions
func (w *MempoolWatcher) fetchAndProcessPendingTransactions(ctx context.Context) error {
	startTime := time.Now()

	// Fetch pending transactions
	var transactions []json.RawMessage
	if err := w.client.CallContext(ctx, &transactions, RPCMethodPendingTransactions); err != nil {
		return err
	}

	// No transactions found
	if len(transactions) == 0 {
		return nil
	}

	fetchTime := time.Since(startTime)
	processStartTime := time.Now()

	processedCount := 0
	duplicateCount := 0
	newTxCount := 0
	updatedTxCount := 0

	// Track transaction hashes found in pendingTransactions for potentially dropped tx validation
	allTxHashes := make(map[string]bool)

	// Process each transaction
	for _, txData := range transactions {
		// Extract hash from transaction data
		var tx struct {
			Hash string `json:"hash"`
		}

		if err := json.Unmarshal(txData, &tx); err != nil {
			w.log.WithError(err).Debug("Failed to parse transaction data from eth_pendingTransactions")
			continue
		}

		txHash := tx.Hash

		// Add to set of known transactions
		allTxHashes[txHash] = true

		// Check if we've already processed this transaction
		w.processedPendingTxsMutex.RLock()
		_, alreadyProcessed := w.processedPendingTxs[txHash]
		w.processedPendingTxsMutex.RUnlock()

		if alreadyProcessed {
			duplicateCount++
			continue
		}

		processedCount++

		// Before adding, check if it would be an update to track metrics properly
		w.pendingTxsMutex.RLock()
		existingTx, exists := w.pendingTxs[txHash]
		isUpdate := exists && existingTx.TxData == nil // It's an update if we had the tx but no data
		w.pendingTxsMutex.RUnlock()

		// Add to pendingTxs using the unified helper
		w.addPendingTransaction(txHash, txData, "eth_pendingTransactions")

		// Track metrics specific to this source
		if isUpdate {
			updatedTxCount++
		} else if !exists {
			newTxCount++
		}
	}

	// Check potentially dropped transactions against what we found in eth_pendingTransactions
	// This is a secondary validation to clear the PotentiallyDropped flag if the tx is still active

	// First collect transactions to update
	txsToClear := make([]string, 0)

	w.pendingTxsMutex.RLock()
	for hash, record := range w.pendingTxs {
		if record.PotentiallyDropped && allTxHashes[hash] {
			txsToClear = append(txsToClear, hash)
		}
	}
	w.pendingTxsMutex.RUnlock()

	// Then update them outside the read lock
	if len(txsToClear) > 0 {
		// Log outside the lock
		for _, hash := range txsToClear {
			w.log.WithField("tx_hash", hash).Debug("Transaction previously marked as dropped was found in eth_pendingTransactions, clearing flag")
		}

		// Now update the flags with a write lock
		w.pendingTxsMutex.Lock()
		for _, hash := range txsToClear {
			// Check again in case the record was removed while we were between locks
			if record, exists := w.pendingTxs[hash]; exists && record.PotentiallyDropped {
				record.PotentiallyDropped = false
			}
		}
		w.pendingTxsMutex.Unlock()
	}

	processTime := time.Since(processStartTime)
	totalTime := time.Since(startTime)

	w.log.WithFields(logrus.Fields{
		"total_txs":     len(transactions),
		"processed_txs": processedCount,
		"duplicate_txs": duplicateCount,
		"new_txs":       newTxCount,
		"updated_txs":   updatedTxCount,
		"fetch_time":    fetchTime.String(),
		"process_time":  processTime.String(),
		"total_time":    totalTime.String(),
	}).Debug("Fetched and processed eth_pendingTransactions")

	return nil
}

// startTransactionProcessor starts a background worker that processes pending transactions in batches
func (w *MempoolWatcher) startTransactionProcessor() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(w.processingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.processPendingTransactions(w.ctx)
			}
		}
	}()
}

// processPendingTransactions processes pending transactions in batches
// First it processes transactions that already have data, then it fetches data for those that don't
func (w *MempoolWatcher) processPendingTransactions(ctx context.Context) {
	startTime := time.Now()

	// Get all pending transactions
	w.pendingTxsMutex.RLock()
	if len(w.pendingTxs) == 0 {
		w.pendingTxsMutex.RUnlock()
		return // Nothing to process
	}

	// Sort transactions into two groups: those with data and those without
	txsWithData := make(map[string]*PendingTxRecord)
	txsWithoutData := make(map[string]*PendingTxRecord)

	for hash, record := range w.pendingTxs {
		if record.TxData != nil {
			txsWithData[hash] = record
		} else {
			txsWithoutData[hash] = record
		}
	}
	w.pendingTxsMutex.RUnlock()

	// Track metrics for the summary log
	withDataCount := len(txsWithData)
	withoutDataCount := len(txsWithoutData)

	// Track results from processing
	withDataProcessed := 0
	withDataFailed := 0
	withoutDataSuccess := 0
	withoutDataNull := 0
	withoutDataFailed := 0

	// Process transactions with data first
	var withDataProcessTime time.Duration
	if withDataCount > 0 {
		withDataStart := time.Now()
		withDataProcessed, withDataFailed = w.processTransactionsWithData(ctx, txsWithData)
		withDataProcessTime = time.Since(withDataStart)
	}

	// Process transactions without data
	var withoutDataProcessTime time.Duration
	if withoutDataCount > 0 {
		withoutDataStart := time.Now()
		withoutDataSuccess, withoutDataNull, withoutDataFailed = w.processTransactionsWithoutData(ctx, txsWithoutData)
		withoutDataProcessTime = time.Since(withoutDataStart)
	}

	totalTime := time.Since(startTime)

	// Add a check to verify the counts balance
	totalWithoutData := withoutDataSuccess + withoutDataNull + withoutDataFailed
	if totalWithoutData != withoutDataCount {
		w.log.WithFields(logrus.Fields{
			"expected_count": withoutDataCount,
			"actual_count":   totalWithoutData,
			"success":        withoutDataSuccess,
			"null":           withoutDataNull,
			"failed":         withoutDataFailed,
		}).Warn("Mismatch in transaction counts")
	}

	// Consolidated log with all information
	w.log.WithFields(logrus.Fields{
		"total_pending":             withDataCount + withoutDataCount,
		"with_data":                 withDataCount,
		"without_data":              withoutDataCount,
		"with_data_processed":       withDataProcessed,
		"with_data_failed":          withDataFailed,
		"with_data_process_time":    withDataProcessTime.String(),
		"without_data_success":      withoutDataSuccess,
		"without_data_null":         withoutDataNull,
		"without_data_failed":       withoutDataFailed,
		"without_data_process_time": withoutDataProcessTime.String(),
		"total_process_time":        totalTime.String(),
	}).Debug("Completed processing pending transactions")
}

// processTransactionsWithData processes transactions that already have data
func (w *MempoolWatcher) processTransactionsWithData(ctx context.Context, txsWithData map[string]*PendingTxRecord) (processed, failed int) {
	// Use atomic variables for safe concurrent updates
	var processedCount, failedCount int32

	// Process transactions in batches
	batchSize := w.batchSize
	if batchSize > len(txsWithData) {
		batchSize = len(txsWithData)
	}

	// Create a bounded work queue with batchSize workers
	sem := make(chan struct{}, batchSize)
	var wg sync.WaitGroup

	// Process each transaction
	for hash, record := range txsWithData {
		// Remove from pendingTxs map
		w.pendingTxsMutex.Lock()
		delete(w.pendingTxs, hash)
		w.pendingTxsMutex.Unlock()

		// Add to processed map
		w.processedPendingTxsMutex.Lock()
		w.processedPendingTxs[hash] = time.Now()
		w.processedPendingTxsMutex.Unlock()

		// Limit concurrency with semaphore
		sem <- struct{}{}
		wg.Add(1)

		// Process the transaction in a separate goroutine
		go func(record *PendingTxRecord) {
			defer func() {
				<-sem // Release semaphore
				wg.Done()
			}()

			if err := w.processTxCallback(ctx, record, record.TxData); err != nil {
				w.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to process transaction with data")
				atomic.AddInt32(&failedCount, 1)
			} else {
				// Update metrics
				w.metricsMutex.Lock()
				w.txsProcessedCount++
				w.txsProcessWithDataCount++
				w.metricsMutex.Unlock()
				atomic.AddInt32(&processedCount, 1)
			}
		}(record)
	}

	// Wait for all processing to complete
	wg.Wait()
	close(sem)

	return int(processedCount), int(failedCount)
}

// processTransactionsWithoutData processes transactions that need data to be fetched
func (w *MempoolWatcher) processTransactionsWithoutData(ctx context.Context, txsWithoutData map[string]*PendingTxRecord) (success, nulls, failed int) {
	// Create batches of transaction hashes to fetch
	var txBatches [][]string
	batchSize := 20 // Optimal batch size for RPC calls
	currentBatch := make([]string, 0, batchSize)

	for hash := range txsWithoutData {
		currentBatch = append(currentBatch, hash)
		if len(currentBatch) >= batchSize {
			txBatches = append(txBatches, currentBatch)
			currentBatch = make([]string, 0, batchSize)
		}
	}

	// Add any remaining transactions
	if len(currentBatch) > 0 {
		txBatches = append(txBatches, currentBatch)
	}

	// Process each batch
	var successCount, nullCount, failCount int32
	var skippedCount int32        // New counter for skipped/missing transactions
	sem := make(chan struct{}, 5) // Limit to 5 concurrent batch requests
	var wg sync.WaitGroup

	for batchIndex, batch := range txBatches {
		sem <- struct{}{}
		wg.Add(1)

		go func(batchNum int, hashes []string) {
			defer func() {
				<-sem // Release semaphore
				wg.Done()
			}()

			// Prepare batch request params
			params := make([]interface{}, len(hashes))
			for i, hash := range hashes {
				params[i] = hash
			}

			// Execute batch request
			results, err := w.client.BatchCallContext(ctx, "eth_getTransactionByHash", params)
			if err != nil {
				w.log.WithError(err).WithField("batch", batchNum).Error("Batch transaction fetch failed")
				atomic.AddInt32(&failCount, int32(len(hashes)))
				return
			}

			// Count for tracking processed transactions in this batch
			processedInBatch := 0

			// Process results
			for i, hash := range hashes {
				if i >= len(results) || len(results[i]) == 0 {
					// Count these as skipped/missing
					atomic.AddInt32(&skippedCount, 1)
					continue
				}

				processedInBatch++

				// Check if result is null
				if string(results[i]) == "null" {
					atomic.AddInt32(&nullCount, 1)
					w.metricsMutex.Lock()
					w.txsNullCount++
					w.metricsMutex.Unlock()

					// Mark this transaction as potentially dropped - we'll do this atomically
					// First check if the record exists
					var shouldMark bool

					w.pendingTxsMutex.RLock()
					record, exists := w.pendingTxs[hash]
					if exists && !record.PotentiallyDropped {
						shouldMark = true
					}
					w.pendingTxsMutex.RUnlock()

					// If needed, log outside the lock (avoiding I/O while holding locks)
					if shouldMark {
						// Then mark as potentially dropped with a brief write lock
						w.pendingTxsMutex.Lock()
						// Double check the record still exists and isn't already marked
						if record, stillExists := w.pendingTxs[hash]; stillExists && !record.PotentiallyDropped {
							record.PotentiallyDropped = true
						}
						w.pendingTxsMutex.Unlock()
					}
					continue
				}

				// Get the record
				record, exists := txsWithoutData[hash]
				if !exists {
					// Count as skipped/missing if record not found
					atomic.AddInt32(&skippedCount, 1)
					continue
				}

				// Update the record with the data
				record.TxData = results[i]

				// Remove from pendingTxs map
				w.pendingTxsMutex.Lock()
				delete(w.pendingTxs, hash)
				w.pendingTxsMutex.Unlock()

				// Add to processed map
				w.processedPendingTxsMutex.Lock()
				w.processedPendingTxs[hash] = time.Now()
				w.processedPendingTxsMutex.Unlock()

				// Process the transaction
				if err := w.processTxCallback(ctx, record, results[i]); err != nil {
					w.log.WithError(err).WithField("tx_hash", hash).Error("Failed to process transaction")
					// Count as failed instead of just logging
					atomic.AddInt32(&failCount, 1)

					// Update global metrics
					w.metricsMutex.Lock()
					w.txsFailedCount++
					w.metricsMutex.Unlock()
				} else {
					// Update metrics
					w.metricsMutex.Lock()
					w.txsProcessedCount++
					w.txsDataFetchedCount++
					w.metricsMutex.Unlock()
					atomic.AddInt32(&successCount, 1)
				}
			}

			// Log if we have missing transactions in this batch
			if processedInBatch < len(hashes) {
				w.log.WithFields(logrus.Fields{
					"batch":     batchNum,
					"expected":  len(hashes),
					"processed": processedInBatch,
					"missing":   len(hashes) - processedInBatch,
				}).Debug("Some transactions were missing from batch results")
			}
		}(batchIndex, batch)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(sem)

	success = int(atomic.LoadInt32(&successCount))
	nulls = int(atomic.LoadInt32(&nullCount))
	failed = int(atomic.LoadInt32(&failCount))
	skipped := int(atomic.LoadInt32(&skippedCount))

	// Debug log to help identify discrepancies
	if skipped > 0 {
		w.log.WithFields(logrus.Fields{
			"total_txs":     len(txsWithoutData),
			"success":       success,
			"null":          nulls,
			"failed":        failed,
			"skipped":       skipped,
			"accounted_for": success + nulls + failed + skipped,
		}).Debug("Transaction processing breakdown")
	}

	// Add skipped transactions to the failed count
	failed += skipped

	return success, nulls, failed
}

// startPruner starts a goroutine that periodically prunes old pending transactions
func (w *MempoolWatcher) startPruner() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(w.pruneDuration / 2) // Run pruner at half the prune duration
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.prunePendingTxs()
				w.pruneProcessedPendingTxs()
			}
		}
	}()
}

// prunePendingTxs removes pending transactions that are older than the prune duration
func (w *MempoolWatcher) prunePendingTxs() {
	w.pendingTxsMutex.Lock()
	defer w.pendingTxsMutex.Unlock()

	now := time.Now()
	prunedCount := 0

	for hash, record := range w.pendingTxs {
		if now.Sub(record.FirstSeen) > w.pruneDuration {
			delete(w.pendingTxs, hash)
			prunedCount++

			// Update metrics
			w.metricsMutex.Lock()
			w.txsExpiredCount++
			w.metricsMutex.Unlock()
		}
	}

	if prunedCount > 0 {
		w.log.WithField("pruned_count", prunedCount).Info("Pruned old pending transactions")
	}
}

// startMetricsLogger starts a goroutine that periodically logs metrics
func (w *MempoolWatcher) startMetricsLogger() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		ticker := time.NewTicker(1 * time.Minute) // Log metrics every minute
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				w.logMetrics()
			}
		}
	}()
}

// logMetrics logs the current metrics
func (w *MempoolWatcher) logMetrics() {
	// Get cumulative metrics (all-time counters)
	w.metricsMutex.RLock()
	cumulative := map[string]int64{
		"txs_total_received":         w.txsReceivedCount,        // All transactions received since start
		"txs_total_processed":        w.txsProcessedCount,       // Successfully processed transactions
		"txs_total_expired":          w.txsExpiredCount,         // Transactions expired due to age
		"txs_total_null":             w.txsNullCount,            // Transactions that returned null from fetch
		"txs_total_with_data":        w.txsWithDataCount,        // Transactions received with data
		"txs_total_data_fetched":     w.txsDataFetchedCount,     // Transactions that needed data to be fetched
		"txs_total_processed_w_data": w.txsProcessWithDataCount, // Transactions processed with data
		"txs_total_failed":           w.txsFailedCount,          // Transactions that failed processing
	}
	w.metricsMutex.RUnlock()

	// Get current state metrics (point-in-time snapshots)
	// Pending transactions
	w.pendingTxsMutex.RLock()
	current := map[string]int{
		"pending_count":             len(w.pendingTxs), // Current count of pending transactions
		"pending_with_data_count":   0,                 // Will be computed below
		"potentially_dropped_count": 0,                 // Will be computed below
	}

	// Source breakdown for current pending transactions
	sourceBreakdown := map[string]int{
		"websocket":               0,
		"txpool_content":          0,
		"eth_pendingTransactions": 0,
	}

	// Count specific attributes of pending transactions
	for _, record := range w.pendingTxs {
		if record.TxData != nil {
			current["pending_with_data_count"]++
		}

		if record.PotentiallyDropped {
			current["potentially_dropped_count"]++
		}

		// Count by source
		if _, ok := sourceBreakdown[record.Source]; ok {
			sourceBreakdown[record.Source]++
		} else {
			// Handle unexpected sources
			sourceBreakdown[record.Source] = 1
		}
	}
	w.pendingTxsMutex.RUnlock()

	// Get processed cache size (current state)
	w.processedPendingTxsMutex.RLock()
	current["processed_cache_size"] = len(w.processedPendingTxs) // Current size of processed tx cache
	w.processedPendingTxsMutex.RUnlock()

	// Calculate derived metrics
	derived := map[string]string{}

	// Calculate rates as percentages
	pendingWithDataRate := 0.0
	if current["pending_count"] > 0 {
		pendingWithDataRate = float64(current["pending_with_data_count"]) / float64(current["pending_count"]) * 100
	}
	derived["pending_with_data_rate"] = fmt.Sprintf("%.2f%%", pendingWithDataRate)

	withDataRate := 0.0
	if cumulative["txs_total_received"] > 0 {
		withDataRate = float64(cumulative["txs_total_with_data"]) / float64(cumulative["txs_total_received"]) * 100
	}
	derived["with_data_rate"] = fmt.Sprintf("%.2f%%", withDataRate)

	successRate := 0.0
	if cumulative["txs_total_received"] > 0 {
		successRate = float64(cumulative["txs_total_processed"]) / float64(cumulative["txs_total_received"]) * 100
	}
	derived["success_rate"] = fmt.Sprintf("%.2f%%", successRate)

	// Get a count of transactions received from the websocket in the last 1 minute
	current["ws_tx_count_last_min"] = w.getRecentWebSocketTxCount(1 * time.Minute)

	// Create a comprehensive log with all relevant information
	logFields := logrus.Fields{}

	// Add cumulative metrics (prefixed with "total_")
	for k, v := range cumulative {
		logFields[k] = v
	}

	// Add current state metrics
	for k, v := range current {
		logFields[k] = v
	}

	// Add source breakdown with clear naming
	logFields["by_source_websocket"] = sourceBreakdown["websocket"]
	logFields["by_source_txpool"] = sourceBreakdown["txpool_content"]
	logFields["by_source_pending_txs"] = sourceBreakdown["eth_pendingTransactions"]

	// Add derived metrics
	for k, v := range derived {
		logFields[k] = v
	}

	w.log.WithFields(logFields).Info("Mempool transaction metrics")
}

// getRecentWebSocketTxCount calculates the number of transactions received via WebSocket
// in the specified duration
func (w *MempoolWatcher) getRecentWebSocketTxCount(duration time.Duration) int {
	count := 0
	cutoff := time.Now().Add(-duration)

	w.pendingTxsMutex.RLock()
	defer w.pendingTxsMutex.RUnlock()

	for _, record := range w.pendingTxs {
		if record.FirstSeen.After(cutoff) && record.Source == "websocket" {
			// Count transactions that came from WebSocket in the specified time window
			count++
		}
	}

	return count
}

// GetMetrics returns the current metrics
func (w *MempoolWatcher) GetMetrics() (totalReceived, totalProcessed, totalExpired, totalNulls int64, pendingCount int, pendingBySource map[string]int) {
	// Get cumulative metrics
	w.metricsMutex.RLock()
	totalReceived = w.txsReceivedCount
	totalProcessed = w.txsProcessedCount
	totalExpired = w.txsExpiredCount
	totalNulls = w.txsNullCount
	w.metricsMutex.RUnlock()

	// Get current stats about pending transactions
	w.pendingTxsMutex.RLock()
	pendingCount = len(w.pendingTxs)

	// Initialize source counts
	pendingBySource = map[string]int{
		"websocket":               0,
		"txpool_content":          0,
		"eth_pendingTransactions": 0,
	}

	// Count transactions by source
	for _, record := range w.pendingTxs {
		if _, ok := pendingBySource[record.Source]; ok {
			pendingBySource[record.Source]++
		} else {
			// Handle unexpected sources
			pendingBySource[record.Source] = 1
		}
	}
	w.pendingTxsMutex.RUnlock()

	return
}

// pruneProcessedPendingTxs removes entries from the processedPendingTxs map that are older than 1 minute
func (w *MempoolWatcher) pruneProcessedPendingTxs() {
	w.processedPendingTxsMutex.Lock()
	defer w.processedPendingTxsMutex.Unlock()

	now := time.Now()
	pruneCutoff := 1 * time.Minute
	prunedCount := 0

	for hash, timestamp := range w.processedPendingTxs {
		if now.Sub(timestamp) > pruneCutoff {
			delete(w.processedPendingTxs, hash)
			prunedCount++
		}
	}

	if prunedCount > 0 && len(w.processedPendingTxs) < 1000 {
		w.log.WithFields(logrus.Fields{
			"pruned_count": prunedCount,
			"remaining":    len(w.processedPendingTxs),
		}).Debug("Pruned processed pending transactions cache")
	}
}

// addPendingTransaction adds a transaction to the pending map if it's not already processed
// or in the pending map. If txData is nil, it's a transaction we only have a hash for.
func (w *MempoolWatcher) addPendingTransaction(txHash string, txData json.RawMessage, source string) {
	// Normalize the hash (ensure it has 0x prefix)
	if len(txHash) >= 2 && txHash[:2] != "0x" {
		txHash = "0x" + txHash
	}

	// Check if we've already processed this transaction
	w.processedPendingTxsMutex.RLock()
	_, alreadyProcessed := w.processedPendingTxs[txHash]
	w.processedPendingTxsMutex.RUnlock()

	if alreadyProcessed {
		// Already processed this transaction, skip it
		return
	}

	// Variables to track state changes
	var flagCleared bool

	// Check if it's already in our pending map
	w.pendingTxsMutex.Lock()
	record, exists := w.pendingTxs[txHash]
	if exists {
		// Already in pending map, but we might need to update the TxData if we didn't have it before
		if record.TxData == nil && txData != nil {
			record.TxData = txData
			// Don't update the source if we're just adding data to an existing transaction
			w.metricsMutex.Lock()
			w.txsWithDataCount++
			w.metricsMutex.Unlock()
		}

		// If this transaction was previously marked as potentially dropped,
		// receiving it again from any source means it's still active
		if record.PotentiallyDropped {
			record.PotentiallyDropped = false
			flagCleared = true
		}
	} else {
		// New transaction, add it to the pending map
		w.pendingTxs[txHash] = &PendingTxRecord{
			Hash:               txHash,
			FirstSeen:          time.Now(),
			Attempts:           0,
			TxData:             txData,
			Source:             source,
			PotentiallyDropped: false, // New transactions are not potentially dropped
		}

		// Update metrics
		w.metricsMutex.Lock()
		w.txsReceivedCount++
		if txData != nil {
			w.txsWithDataCount++
		}
		w.metricsMutex.Unlock()
	}
	w.pendingTxsMutex.Unlock()

	// Log outside the lock
	if flagCleared {
		w.log.WithFields(logrus.Fields{
			"tx_hash": txHash,
			"source":  source,
		}).Debug("Cleared potentially dropped flag for transaction still active in the network")
	}

	// Debug logging for new transactions is selectively disabled to avoid log flooding
	// if source != "websocket" {
	//     w.log.WithFields(logrus.Fields{
	//         "tx_hash":  txHash,
	//         "has_data": txData != nil,
	//         "source":   source,
	//     }).Debug("Added new pending transaction")
	// }
}
