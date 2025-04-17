package execution

import (
	"encoding/json"
	"sync"
	"time"
)

// txQueueItem represents an item in the transaction processing queue.
type txQueueItem struct {
	record *PendingTxRecord
	txData json.RawMessage // May be nil if we only have the hash.
}

// txQueue is a thread-safe queue for transaction processing.
type txQueue struct {
	items     chan txQueueItem
	processed map[string]time.Time // Map of processed transaction hashes to when they were processed.
	mutex     sync.RWMutex         // Mutex for the processed map.
	pruneDur  time.Duration        // How long to keep processed transactions in memory.
	capacity  int                  // Store capacity for metric calculations.
	metrics   *Metrics             // Metrics for tracking queue stats.
}

// newTxQueue creates a new transaction queue.
func newTxQueue(metrics *Metrics, capacity int, pruneDuration time.Duration) *txQueue {
	return &txQueue{
		items:     make(chan txQueueItem, capacity),
		processed: make(map[string]time.Time),
		mutex:     sync.RWMutex{},
		pruneDur:  pruneDuration,
		capacity:  capacity,
		metrics:   metrics,
	}
}

// updateQueueMetrics updates the queue size metrics.
func (q *txQueue) updateQueueMetrics() {
	// Update current queue size.
	queueSize := len(q.items)
	q.metrics.SetQueueSize(queueSize)
}

// add adds a transaction to the queue if not already processed
// returns true if the transaction was added, false if it was already processed.
func (q *txQueue) add(record *PendingTxRecord, txData json.RawMessage) bool {
	// Check if already processed
	q.mutex.RLock()
	_, exists := q.processed[record.Hash]
	q.mutex.RUnlock()
	if exists {
		return false
	}

	// Try to add to queue.
	select {
	case q.items <- txQueueItem{record: record, txData: txData}:
		// Get queue size after adding.
		queueSize := len(q.items)

		// Update current queue size.
		q.metrics.SetQueueSize(queueSize)

		// Update throughput counter.
		q.metrics.AddQueueThroughput(1)

		return true
	default:
		// Queue is full.
		q.metrics.AddQueueRejections(1)
		return false
	}
}

// markProcessed marks a transaction as processed.
func (q *txQueue) markProcessed(hash string) {
	q.mutex.Lock()
	q.processed[hash] = time.Now()
	q.mutex.Unlock()

	// Update metrics if available.
	q.metrics.SetProcessedCacheSize(len(q.processed))

	// Update queue metrics after processing.
	q.updateQueueMetrics()
}

// isProcessed checks if a transaction has been processed.
func (q *txQueue) isProcessed(hash string) bool {
	q.mutex.RLock()
	_, exists := q.processed[hash]
	q.mutex.RUnlock()
	return exists
}

// prune removes processed transactions older than the prune duration
// This is different from the main pending transaction pruning, as processed
// transactions only need to be kept long enough to prevent duplicate processing.
func (q *txQueue) prune() int {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	now := time.Now()
	prunedCount := 0

	for hash, timestamp := range q.processed {
		if now.Sub(timestamp) > q.pruneDur {
			delete(q.processed, hash)
			prunedCount++
		}
	}

	// Update metrics if available.
	if prunedCount > 0 {
		q.metrics.SetProcessedCacheSize(len(q.processed))
	}

	return prunedCount
}

// markTxReceived tracks a transaction that was received, tracking its source.
func (q *txQueue) markTxReceived(source string) {
	q.metrics.AddMempoolTxReceived(1)
	q.metrics.AddTxBySource(source, 1)
}

// recordProcessingOutcome records the outcome of processing a transaction.
func (q *txQueue) recordProcessingOutcome(outcome string, count int) {
	q.metrics.AddTxProcessingOutcome(outcome, count)
}

// setTxPending sets the current count of pending transactions.
func (q *txQueue) setTxPending(count int) {
	q.metrics.SetMempoolTxPending(count)
}
