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
	Hash      string
	FirstSeen time.Time
	Attempts  int
}

// MempoolWatcher watches for pending transactions using a two-phase approach:
// 1. Subscribe to newPendingTransactions via websocket to get notified of new txs
// 2. Periodically fetch txpool_content to get full transaction details
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

	// Callback when a transaction is found in txpool_content
	processTxCallback func(context.Context, *PendingTxRecord, json.RawMessage) error

	// Metrics
	txsReceivedCount  int64
	txsProcessedCount int64
	txsExpiredCount   int64
	txsNullCount      int64 // Count of transactions that returned null from backfill
	metricsMutex      sync.RWMutex

	// Backfill state
	isBackfilling bool
	backfillMutex sync.Mutex

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
	ctx, cancel := context.WithCancel(context.Background())

	return &MempoolWatcher{
		client:                   client,
		log:                      log.WithField("component", "execution/mempool_watcher"),
		pendingTxs:               make(map[string]*PendingTxRecord),
		pendingTxsMutex:          sync.RWMutex{},
		wg:                       sync.WaitGroup{},
		ctx:                      ctx,
		cancel:                   cancel,
		fetchInterval:            fetchInterval,
		pruneDuration:            pruneDuration,
		processTxCallback:        processTxCallback,
		txsReceivedCount:         0,
		txsProcessedCount:        0,
		txsExpiredCount:          0,
		txsNullCount:             0,
		metricsMutex:             sync.RWMutex{},
		isBackfilling:            false,
		backfillMutex:            sync.Mutex{},
		processedPendingTxs:      make(map[string]time.Time),
		processedPendingTxsMutex: sync.RWMutex{},
	}
}

// Start starts the mempool watcher
func (w *MempoolWatcher) Start(ctx context.Context) error {
	// Start WebSocket subscription for new pending transactions
	if err := w.startNewPendingTxSubscription(); err != nil {
		return fmt.Errorf("failed to start WebSocket subscription: %w", err)
	}

	// Start periodic fetching of txpool content
	w.startPeriodicFetcher()

	// Start periodic metrics logging
	w.startMetricsLogger()

	// Start periodic pruning of old pending txs
	//w.startPruner()

	// Start supplementary pending transaction fetcher
	w.startPendingTransactionsFetcher()

	// Start background transaction backfiller
	//w.startBackfiller()

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
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			<-w.ctx.Done()
			cancel()
		}()

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
	// Create channel to receive transaction hashes
	txChan := make(chan string)

	// Setup statistics ticker
	statsTicker := time.NewTicker(60 * time.Second)

	var (
		txCounter   int
		txCounterMu sync.Mutex
	)

	// Start stats goroutine
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		defer statsTicker.Stop() // Move defer inside the goroutine

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-statsTicker.C:
				txCounterMu.Lock()
				count := txCounter
				txCounter = 0 // Reset counter
				txCounterMu.Unlock()

				if count > 0 {
					w.log.WithFields(logrus.Fields{
						"method":   "websocket_subscribe",
						"tx_count": count,
					}).Info("New pending transactions from mempool in last 60s")
				}
			}
		}
	}()

	// Subscribe to pending transactions
	var err error
	w.wsSubscription, err = w.client.wsClient.EthSubscribe(w.ctx, txChan, string(SubNewPendingTransactions))
	if err != nil {
		return fmt.Errorf("failed to subscribe to %s: %w", SubNewPendingTransactions, err)
	}

	// Process incoming transaction hashes
	w.wg.Add(1)
	go func() {
		defer w.wg.Done()

		for {
			select {
			case <-w.ctx.Done():
				return
			case txHash, ok := <-txChan:
				if !ok {
					w.log.Error("Transaction channel closed unexpectedly")

					return
				}

				// Increment counter for stats
				txCounterMu.Lock()
				txCounter++
				txCounterMu.Unlock()

				// Process the transaction hash
				if err := w.handleNewPendingTransaction(w.ctx, txHash); err != nil {
					w.log.WithError(err).WithField("tx_hash", txHash).
						Error("Failed to handle new pending transaction")
				}
			}
		}
	}()

	return nil
}

// handleNewPendingTransaction is called when a new pending transaction hash is received
func (w *MempoolWatcher) handleNewPendingTransaction(ctx context.Context, txHash string) error {
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
		return nil
	}

	// Mark as processed
	w.processedPendingTxsMutex.Lock()
	w.processedPendingTxs[txHash] = time.Now()
	w.processedPendingTxsMutex.Unlock()

	w.pendingTxsMutex.Lock()
	defer w.pendingTxsMutex.Unlock()

	// Only add if we haven't seen this transaction before
	if _, exists := w.pendingTxs[txHash]; !exists {
		w.pendingTxs[txHash] = &PendingTxRecord{
			Hash:      txHash,
			FirstSeen: time.Now(),
			Attempts:  0,
		}

		// Update metrics
		w.metricsMutex.Lock()
		w.txsReceivedCount++
		w.metricsMutex.Unlock()
	}

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

// fetchAndProcessTxPool fetches txpool_content and processes transactions
func (w *MempoolWatcher) fetchAndProcessTxPool(ctx context.Context) error {
	startTime := time.Now()

	// Fetch txpool content
	var result struct {
		Pending map[string]map[string]json.RawMessage `json:"pending"`
	}

	if err := w.client.CallContext(ctx, &result, RPCMethodTxpoolContent); err != nil {
		fetchTime := time.Since(startTime)
		w.log.WithFields(logrus.Fields{
			"error":      err.Error(),
			"fetch_time": fetchTime.String(),
		}).Error("Failed to fetch txpool content")
		return fmt.Errorf("failed to fetch txpool content: %w", err)
	}

	fetchTime := time.Since(startTime)
	processStartTime := time.Now()

	// Prune old entries from the processed pending transactions cache
	w.pruneProcessedPendingTxs()

	// Increment attempt counter for all pending transactions
	w.pendingTxsMutex.Lock()
	pendingCount := len(w.pendingTxs)
	for _, record := range w.pendingTxs {
		record.Attempts++
	}
	w.pendingTxsMutex.Unlock()

	// Process each transaction in the txpool
	matchedCount := 0
	duplicateCount := 0
	totalTxCount := 0

	for _, accounts := range result.Pending {
		for _, txData := range accounts {
			totalTxCount++

			// Extract hash from transaction data
			var tx struct {
				Hash string `json:"hash"`
			}

			if err := json.Unmarshal(txData, &tx); err != nil {
				w.log.WithError(err).Debug("Failed to parse transaction data")
				continue
			}

			txHash := tx.Hash

			// Check if we've already processed this transaction through any method
			w.processedPendingTxsMutex.RLock()
			_, alreadyProcessed := w.processedPendingTxs[txHash]
			w.processedPendingTxsMutex.RUnlock()

			if alreadyProcessed {
				duplicateCount++
				continue
			}

			// Mark as processed to avoid duplicates across all methods
			w.processedPendingTxsMutex.Lock()
			w.processedPendingTxs[txHash] = time.Now()
			w.processedPendingTxsMutex.Unlock()

			// Check if this is a transaction we're waiting for
			w.pendingTxsMutex.Lock()
			record, exists := w.pendingTxs[txHash]

			if exists {
				matchedCount++

				// Remove from pending map to avoid processing it again
				delete(w.pendingTxs, txHash)
				w.pendingTxsMutex.Unlock()

				// Process the transaction (in a separate goroutine to avoid blocking)
				go func(record *PendingTxRecord, txData json.RawMessage) {
					if err := w.processTxCallback(ctx, record, txData); err != nil {
						w.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to process transaction")
					} else {
						// Update metrics
						w.metricsMutex.Lock()
						w.txsProcessedCount++
						w.metricsMutex.Unlock()
					}
				}(record, txData)
			} else {
				// This is a new transaction not in our pending map
				// We'll create a record and process it directly
				record := &PendingTxRecord{
					Hash:      txHash,
					FirstSeen: time.Now(),
					Attempts:  0,
				}

				w.pendingTxsMutex.Unlock()

				// Update metrics for a new transaction
				w.metricsMutex.Lock()
				w.txsReceivedCount++
				w.metricsMutex.Unlock()

				// Process the transaction directly
				go func(record *PendingTxRecord, txData json.RawMessage) {
					if err := w.processTxCallback(ctx, record, txData); err != nil {
						w.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to process new transaction from txpool_content")
					} else {
						// Update metrics
						w.metricsMutex.Lock()
						w.txsProcessedCount++
						w.metricsMutex.Unlock()
					}
				}(record, txData)
			}
		}
	}

	processTime := time.Since(processStartTime)
	totalTime := time.Since(startTime)

	w.log.WithFields(logrus.Fields{
		"pending_txs":   pendingCount,
		"txpool_txs":    totalTxCount,
		"matched_txs":   matchedCount,
		"duplicate_txs": duplicateCount,
		"fetch_time":    fetchTime.String(),
		"process_time":  processTime.String(),
		"total_time":    totalTime.String(),
	}).Debug("Fetched and processed txpool content")

	return nil
}

// scheduleBackfill adds pending transactions to the backfill queue
func (w *MempoolWatcher) scheduleBackfill(ctx context.Context) {
	// Prevent multiple backfills from running concurrently
	w.backfillMutex.Lock()
	if w.isBackfilling {
		w.backfillMutex.Unlock()

		return
	}
	w.isBackfilling = true
	w.backfillMutex.Unlock()

	const maxBackfillBatchSize = 1000 // Maximum transactions to schedule at once

	// Create a copy of eligible transactions to backfill
	w.pendingTxsMutex.RLock()
	eligibleCount := 0
	eligibleTxs := make([]string, 0, maxBackfillBatchSize)

	for hash, record := range w.pendingTxs {
		if time.Since(record.FirstSeen) > 5*time.Second && record.Attempts >= 1 {
			eligibleTxs = append(eligibleTxs, hash)
			eligibleCount++

			if eligibleCount >= maxBackfillBatchSize {
				break
			}
		}
	}
	pendingCount := len(w.pendingTxs)
	w.pendingTxsMutex.RUnlock()

	if eligibleCount == 0 {
		// Reset backfill flag
		w.backfillMutex.Lock()
		w.isBackfilling = false
		w.backfillMutex.Unlock()
		return
	}

	w.log.WithFields(logrus.Fields{
		"backfill_count": eligibleCount,
		"pending_txs":    pendingCount,
	}).Debug("Scheduling mempool transaction backfill")

	// Process transactions in small batches with workers to avoid overwhelming the client
	go w.processBackfillTransactions(ctx, eligibleTxs)
}

// processBackfillTransactions processes transactions in batches using JSON-RPC batch calls
func (w *MempoolWatcher) processBackfillTransactions(ctx context.Context, txHashes []string) {
	// Make sure to mark backfill as complete when done
	defer func() {
		w.backfillMutex.Lock()
		w.isBackfilling = false
		w.backfillMutex.Unlock()
	}()

	const (
		batchSize     = 50                     // Number of transactions per JSON-RPC batch
		maxConcurrent = 2                      // Number of concurrent batch requests
		rateLimitWait = 500 * time.Millisecond // Time to wait between batches
	)

	startTime := time.Now()

	// Track success and failure counts
	var (
		successCount int32
		failCount    int32
		nullCount    int32
		skipCount    int32
	)

	// Group transactions into batches for batch processing
	var batches [][]string
	for i := 0; i < len(txHashes); i += batchSize {
		end := i + batchSize
		if end > len(txHashes) {
			end = len(txHashes)
		}
		batches = append(batches, txHashes[i:end])
	}

	// Process batches with limited concurrency
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for batchIndex, batchHashes := range batches {
		// Filter out transactions that have already been processed
		var filteredHashes []string
		var filteredRecords map[string]*PendingTxRecord = make(map[string]*PendingTxRecord)

		// Check if any transactions in this batch have already been processed
		w.processedPendingTxsMutex.RLock()
		for _, hash := range batchHashes {
			if _, processed := w.processedPendingTxs[hash]; !processed {
				filteredHashes = append(filteredHashes, hash)
			} else {
				atomic.AddInt32(&skipCount, 1)
			}
		}
		w.processedPendingTxsMutex.RUnlock()

		if len(filteredHashes) == 0 {
			continue // All transactions in this batch were already processed
		}

		// Get records from map (under lock)
		w.pendingTxsMutex.RLock()
		batchParams := make([]interface{}, 0, len(filteredHashes))

		for _, hash := range filteredHashes {
			if record, exists := w.pendingTxs[hash]; exists {
				filteredRecords[hash] = record
				batchParams = append(batchParams, hash)
			}
		}
		w.pendingTxsMutex.RUnlock()

		if len(batchParams) == 0 {
			continue // All transactions in this batch were already processed or not found
		}

		// Limit concurrency with semaphore
		sem <- struct{}{}
		wg.Add(1)

		go func(batchNum int, params []interface{}, txRecords map[string]*PendingTxRecord) {
			defer func() {
				<-sem // Release semaphore
				wg.Done()
			}()

			// Rate limiting delay between batches
			time.Sleep(rateLimitWait)

			// Use a timeout for batch request
			fetchCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			// Execute batch request.
			results, err := w.client.BatchCallContext(fetchCtx, "eth_getTransactionByHash", params)
			if err != nil {
				w.log.WithError(err).WithField("batch", batchNum).Error("Batch request failed")
				atomic.AddInt32(&failCount, int32(len(params)))
				return
			}

			// Process each result
			for i, param := range params {
				txHash := param.(string)

				// Skip empty results
				if i >= len(results) || len(results[i]) == 0 {
					continue
				}

				// eth_getTransactionByHash doesn't fetch from the mempool.
				// - If the transaction is already in a block, it returns full transaction details
				// - If the transaction is still pending (in the mempool), it returns null. These will be picked-up by our processing of txpool_content.
				if string(results[i]) == "null" {
					// Remove from pending map, even if it's null - we don't want to keep trying
					// w.pendingTxsMutex.Lock()
					// delete(w.pendingTxs, txHash)
					// w.pendingTxsMutex.Unlock()

					// Update metrics for null transactions
					w.metricsMutex.Lock()
					w.txsNullCount++
					w.metricsMutex.Unlock()

					atomic.AddInt32(&nullCount, 1)
					continue
				}

				// Get the corresponding record - important to check existence again here, incase the tx was removed from the pending map
				// by a parallel process (eg, txpool_content mapping).
				record, exists := txRecords[txHash]
				if !exists {
					continue
				}

				// Remove from pending map
				w.pendingTxsMutex.Lock()
				delete(w.pendingTxs, txHash)
				w.pendingTxsMutex.Unlock()

				// Mark as processed in the global tracking map to avoid duplicates
				w.processedPendingTxsMutex.Lock()
				w.processedPendingTxs[txHash] = time.Now()
				w.processedPendingTxsMutex.Unlock()

				// Process the transaction
				if err := w.processTxCallback(ctx, record, results[i]); err != nil {
					w.log.WithError(err).WithField("tx_hash", txHash).Error("Failed to process backfilled transaction")
					atomic.AddInt32(&failCount, 1)
				} else {
					// Update metrics
					w.metricsMutex.Lock()
					w.txsProcessedCount++
					w.metricsMutex.Unlock()
					atomic.AddInt32(&successCount, 1)
				}
			}

			// Count null responses separately for logging
			nullTxs := 0
			for _, result := range results {
				if string(result) == "null" {
					nullTxs++
				}
			}

			w.log.WithFields(logrus.Fields{
				"batch":     batchNum,
				"processed": len(results),
				"null_txs":  nullTxs,
				"valid_txs": len(results) - nullTxs,
				"total":     len(params),
			}).Debug("Backfilled mempool transaction batch")

		}(batchIndex, batchParams, filteredRecords)
	}

	// Wait for all batches to complete
	wg.Wait()
	close(sem)

	processTime := time.Since(startTime)

	success := int(atomic.LoadInt32(&successCount))
	failed := int(atomic.LoadInt32(&failCount))
	nulls := int(atomic.LoadInt32(&nullCount))
	skipped := int(atomic.LoadInt32(&skipCount))

	if success > 0 || failed > 0 || nulls > 0 || skipped > 0 {
		w.log.WithFields(logrus.Fields{
			"success_count": success,
			"failed_count":  failed,
			"null_count":    nulls,
			"skipped_count": skipped,
			"process_time":  processTime.String(),
			"tx_count":      len(txHashes),
		}).Debug("Completed mempool transaction backfill")
	}
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
	w.metricsMutex.RLock()
	received := w.txsReceivedCount
	processed := w.txsProcessedCount
	expired := w.txsExpiredCount
	nulls := w.txsNullCount
	w.metricsMutex.RUnlock()

	w.pendingTxsMutex.RLock()
	pendingCount := len(w.pendingTxs)
	w.pendingTxsMutex.RUnlock()

	// Calculate success rate including null transactions as "processed"
	// since they were handled and removed from the pending queue
	totalHandled := processed + nulls
	successRate := calculatePercentage(totalHandled, received)

	w.log.WithFields(logrus.Fields{
		"txs_received":  received,
		"txs_processed": processed,
		"txs_expired":   expired,
		"txs_null":      nulls,
		"total_handled": totalHandled,
		"pending_txs":   pendingCount,
		"success_rate":  fmt.Sprintf("%.2f%%", successRate),
	}).Info("Mempool watcher metrics")
}

// GetMetrics returns the current metrics
func (w *MempoolWatcher) GetMetrics() (received, processed, expired, nulls int64, pendingCount int) {
	w.metricsMutex.RLock()
	received = w.txsReceivedCount
	processed = w.txsProcessedCount
	expired = w.txsExpiredCount
	nulls = w.txsNullCount
	w.metricsMutex.RUnlock()

	w.pendingTxsMutex.RLock()
	pendingCount = len(w.pendingTxs)
	w.pendingTxsMutex.RUnlock()

	return
}

// calculatePercentage calculates the percentage of a out of b
func calculatePercentage(a, b int64) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}

// startBackfiller starts a goroutine that periodically processes pending transactions
// that weren't found in txpool_content but might still be available via RPC
func (w *MempoolWatcher) startBackfiller() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		// Use a different interval from the fetcher to avoid contention
		// The backfiller can run less frequently than the fetcher
		backfillInterval := w.fetchInterval / 2

		ticker := time.NewTicker(backfillInterval)
		defer ticker.Stop()

		for {
			select {
			case <-w.ctx.Done():
				return
			case <-ticker.C:
				// Check if we have pending transactions to process
				w.pendingTxsMutex.RLock()
				pendingCount := len(w.pendingTxs)
				w.pendingTxsMutex.RUnlock()

				if pendingCount > 0 {
					w.scheduleBackfill(w.ctx)
				}
			}
		}
	}()
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

// fetchAndProcessPendingTransactions fetches pending transactions using eth_pendingTransactions
func (w *MempoolWatcher) fetchAndProcessPendingTransactions(ctx context.Context) error {
	startTime := time.Now()

	// Fetch pending transactions
	var transactions []json.RawMessage

	if err := w.client.CallContext(ctx, &transactions, string(RPCMethodPendingTransactions)); err != nil {
		fetchTime := time.Since(startTime)
		w.log.WithFields(logrus.Fields{
			"error":      err.Error(),
			"fetch_time": fetchTime.String(),
		}).Error("Failed to fetch pending transactions")
		return fmt.Errorf("failed to fetch pending transactions: %w", err)
	}

	fetchTime := time.Since(startTime)
	processStartTime := time.Now()

	// Prune old entries from the processed pending transactions cache
	w.pruneProcessedPendingTxs()

	// Process each transaction
	processedCount := 0
	newTxCount := 0
	duplicateCount := 0

	for _, txData := range transactions {
		// Extract hash from transaction data
		var tx struct {
			Hash string `json:"hash"`
		}

		if err := json.Unmarshal(txData, &tx); err != nil {
			w.log.WithError(err).Debug("Failed to parse transaction data")
			continue
		}

		txHash := tx.Hash
		if txHash == "" {
			continue
		}

		// Check if we've already processed this transaction recently via eth_pendingTransactions
		w.processedPendingTxsMutex.RLock()
		_, alreadyProcessed := w.processedPendingTxs[txHash]
		w.processedPendingTxsMutex.RUnlock()

		if alreadyProcessed {
			duplicateCount++
			continue
		}

		processedCount++

		// Mark as processed to avoid duplicates in future calls
		w.processedPendingTxsMutex.Lock()
		w.processedPendingTxs[txHash] = time.Now()
		w.processedPendingTxsMutex.Unlock()

		// Check if we've already seen this transaction via WebSocket
		w.pendingTxsMutex.Lock()
		record, exists := w.pendingTxs[txHash]

		if exists {
			// We've seen this before via WebSocket, remove it and process it
			delete(w.pendingTxs, txHash)
			w.pendingTxsMutex.Unlock()

			// Process the transaction
			go func(record *PendingTxRecord, txData json.RawMessage) {
				if err := w.processTxCallback(ctx, record, txData); err != nil {
					w.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to process transaction from eth_pendingTransactions")
				} else {
					// Update metrics
					w.metricsMutex.Lock()
					w.txsProcessedCount++
					w.metricsMutex.Unlock()
				}
			}(record, txData)
		} else {
			// This is a new transaction we haven't seen via WebSocket
			w.pendingTxsMutex.Unlock()

			record := &PendingTxRecord{
				Hash:      txHash,
				FirstSeen: time.Now(),
				Attempts:  0,
			}

			// Update metrics for a new transaction
			w.metricsMutex.Lock()
			w.txsReceivedCount++
			w.metricsMutex.Unlock()

			newTxCount++

			// Process the transaction
			go func(record *PendingTxRecord, txData json.RawMessage) {
				if err := w.processTxCallback(ctx, record, txData); err != nil {
					w.log.WithError(err).WithField("tx_hash", record.Hash).Error("Failed to process new transaction from eth_pendingTransactions")
				} else {
					// Update metrics
					w.metricsMutex.Lock()
					w.txsProcessedCount++
					w.metricsMutex.Unlock()
				}
			}(record, txData)
		}
	}

	processTime := time.Since(processStartTime)
	totalTime := time.Since(startTime)

	w.log.WithFields(logrus.Fields{
		"total_txs":     len(transactions),
		"processed_txs": processedCount,
		"duplicate_txs": duplicateCount,
		"new_txs":       newTxCount,
		"fetch_time":    fetchTime.String(),
		"process_time":  processTime.String(),
		"total_time":    totalTime.String(),
	}).Debug("Fetched and processed eth_pendingTransactions")

	return nil
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
