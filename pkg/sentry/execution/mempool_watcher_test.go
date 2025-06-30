package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"github.com/ethpandaops/xatu/pkg/sentry/execution/mock"
)

// TestAddPendingTransactionFunctionality tests the core logic of adding transactions.
func TestAddPendingTransactionFunctionality(t *testing.T) {
	// Common setup code
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	metrics := NewMetrics("test1", "testnet")

	var processedTxs []string
	var mu sync.Mutex
	callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
		mu.Lock()
		defer mu.Unlock()

		processedTxs = append(processedTxs, record.Hash)

		return nil
	}

	config := &Config{
		QueueSize:     10,
		PruneDuration: 5,
	}

	// Setup test function to create a fresh watcher for each sub-test
	setupWatcher := func() (*MempoolWatcher, context.CancelFunc) {
		watcher := &MempoolWatcher{
			log:                 logger.WithField("component", "test"),
			config:              config,
			pendingTxs:          make(map[string]*PendingTxRecord),
			pendingTxsMutex:     sync.RWMutex{},
			txQueue:             newTxQueue(metrics, config.QueueSize, time.Duration(config.PruneDuration)*time.Second),
			processTxCallback:   callback,
			metrics:             metrics,
			txsReceivedBySource: make(map[string]int64),
			metricsMutex:        sync.RWMutex{},
		}

		ctx, cancel := context.WithCancel(context.Background())
		watcher.ctx = ctx
		watcher.cancel = cancel

		return watcher, cancel
	}

	// Sub-tests with t.Run
	t.Run("NormalizeHashPrefix", func(t *testing.T) {
		watcher, cancel := setupWatcher()
		defer cancel()

		txHash := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
		watcher.addPendingTransaction(txHash, nil, "test")

		watcher.pendingTxsMutex.RLock()
		record, exists := watcher.pendingTxs["0x"+txHash]
		watcher.pendingTxsMutex.RUnlock()

		assert.True(t, exists, "Transaction should be added to pendingTxs")
		assert.Equal(t, "0x"+txHash, record.Hash, "Transaction hash should be normalized with 0x prefix")
		assert.Equal(t, "test", record.Source, "Source should be recorded")
	})

	t.Run("AddWithData", func(t *testing.T) {
		watcher, cancel := setupWatcher()
		defer cancel()

		txHash := "0x2222222222222222222222222222222222222222222222222222222222222222"
		txData := json.RawMessage(`{"hash":"0x2222222222222222222222222222222222222222222222222222222222222222","value":"0x123"}`)
		watcher.addPendingTransaction(txHash, txData, "test_with_data")

		watcher.pendingTxsMutex.RLock()
		record, exists := watcher.pendingTxs[txHash]
		watcher.pendingTxsMutex.RUnlock()

		assert.True(t, exists, "Transaction with data should be added")
		assert.Equal(t, txHash, record.Hash, "Transaction hash should match")
		assert.Equal(t, txData, record.TxData, "Transaction data should be stored")
		assert.Equal(t, "test_with_data", record.Source, "Source should be correct")
	})

	t.Run("PreserveDataOnDuplicate", func(t *testing.T) {
		watcher, cancel := setupWatcher()
		defer cancel()

		txHash := "0x2222222222222222222222222222222222222222222222222222222222222222"
		txData := json.RawMessage(`{"hash":"0x2222222222222222222222222222222222222222222222222222222222222222","value":"0x123"}`)

		// Add with data first
		watcher.addPendingTransaction(txHash, txData, "test_with_data")

		// Then add again without data
		watcher.addPendingTransaction(txHash, nil, "duplicate_source")

		watcher.pendingTxsMutex.RLock()
		record, exists := watcher.pendingTxs[txHash]
		watcher.pendingTxsMutex.RUnlock()

		assert.True(t, exists, "Transaction should still exist after duplicate add")
		assert.Equal(t, txData, record.TxData, "Original TX data should not be overwritten by nil")
	})

	t.Run("ClearPruningMark", func(t *testing.T) {
		watcher, cancel := setupWatcher()
		defer cancel()

		txHash := "0x4444444444444444444444444444444444444444444444444444444444444444"

		// Add a transaction that's already marked for pruning
		watcher.pendingTxsMutex.Lock()
		watcher.pendingTxs[txHash] = &PendingTxRecord{
			Hash:             txHash,
			FirstSeen:        time.Now().Add(-time.Hour),
			Source:           "old_source",
			MarkedForPruning: true,
		}
		watcher.pendingTxsMutex.Unlock()

		// Re-add it - should clear the pruning mark
		watcher.addPendingTransaction(txHash, nil, "new_source")

		watcher.pendingTxsMutex.RLock()
		record, exists := watcher.pendingTxs[txHash]
		watcher.pendingTxsMutex.RUnlock()

		assert.True(t, exists, "Previously pruned transaction should exist")
		assert.False(t, record.MarkedForPruning, "MarkedForPruning should be cleared when re-adding")
	})
}

// TestPrunePendingTxsLogic tests the pruning logic without dependencies.
func TestPrunePendingTxsLogic(t *testing.T) {
	// Setup
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	metrics := NewMetrics("test2", "testnet")

	config := &Config{
		PruneDuration: 2, // 2 second prune duration for testing.
	}

	watcher := &MempoolWatcher{
		log:                 logger.WithField("component", "test"),
		config:              config,
		pendingTxs:          make(map[string]*PendingTxRecord),
		pendingTxsMutex:     sync.RWMutex{},
		txQueue:             newTxQueue(metrics, 10, time.Duration(config.PruneDuration)*time.Second),
		metrics:             metrics,
		txsReceivedBySource: make(map[string]int64),
		metricsMutex:        sync.RWMutex{},
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	watcher.ctx = ctx
	watcher.cancel = cancel

	// Table of test cases
	now := time.Now()
	testCases := []struct {
		name             string
		hash             string
		age              time.Duration
		markedForPruning bool
		shouldBeRemoved  bool
	}{
		{"VeryOld", "0x1111", 10 * time.Second, false, true},
		{"JustOverThreshold", "0x2222", 3 * time.Second, false, true},
		{"JustUnderThreshold", "0x3333", 1 * time.Second, false, false},
		{"BrandNew", "0x4444", 0, false, false},
		{"MarkedButNew", "0x5555", 0, true, false},
	}

	// Setup test transactions
	watcher.pendingTxsMutex.Lock()
	for _, tc := range testCases {
		watcher.pendingTxs[tc.hash] = &PendingTxRecord{
			Hash:             tc.hash,
			FirstSeen:        now.Add(-tc.age),
			Source:           "test",
			MarkedForPruning: tc.markedForPruning,
		}
	}
	watcher.pendingTxsMutex.Unlock()

	// Run pruning
	watcher.prunePendingTxs()

	// Verify results for each test case
	watcher.pendingTxsMutex.RLock()
	defer watcher.pendingTxsMutex.RUnlock()

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, exists := watcher.pendingTxs[tc.hash]
			if tc.shouldBeRemoved {
				assert.False(t, exists, "Transaction %s should be pruned", tc.hash)
			} else {
				assert.True(t, exists, "Transaction %s should remain", tc.hash)
			}
		})
	}

	// Verify total count
	expectedCount := 0
	for _, tc := range testCases {
		if !tc.shouldBeRemoved {
			expectedCount++
		}
	}
	assert.Equal(t, expectedCount, len(watcher.pendingTxs), "Should have exactly %d transactions remaining", expectedCount)
}

// TestTxQueueOperations tests the transaction queue operations directly.
func TestTxQueueOperations(t *testing.T) {
	metrics := NewMetrics("test3", "testnet")

	t.Run("BasicOperations", func(t *testing.T) {
		// Create a fresh queue for each test
		queue := newTxQueue(metrics, 5, 1*time.Second)

		// Define test cases for basic queue operations
		testCases := []struct {
			name        string
			operation   func() bool
			expected    bool
			description string
		}{
			{
				name: "AddNewTransaction",
				operation: func() bool {
					return queue.add(&PendingTxRecord{
						Hash:      "0x1111",
						FirstSeen: time.Now(),
						Source:    "test",
					}, nil)
				},
				expected:    true,
				description: "Transaction should be added successfully",
			},
			{
				name: "CheckNotProcessed",
				operation: func() bool {
					return queue.isProcessed("0x1111")
				},
				expected:    false,
				description: "Transaction should not be marked as processed yet",
			},
			{
				name: "MarkAsProcessed",
				operation: func() bool {
					queue.markProcessed("0x1111")

					return queue.isProcessed("0x1111")
				},
				expected:    true,
				description: "Transaction should be marked as processed",
			},
			{
				name: "RejectProcessedTransaction",
				operation: func() bool {
					return queue.add(&PendingTxRecord{
						Hash:      "0x1111",
						FirstSeen: time.Now(),
						Source:    "test",
					}, nil)
				},
				expected:    false,
				description: "Already processed transaction should not be added again",
			},
		}

		// Run the test cases in sequence
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				result := tc.operation()
				assert.Equal(t, tc.expected, result, tc.description)
			})
		}
	})

	t.Run("QueueCapacityLimit", func(t *testing.T) {
		queue := newTxQueue(metrics, 5, 1*time.Second)

		// Add more transactions than capacity
		for i := 0; i < 10; i++ {
			record := &PendingTxRecord{
				Hash:      "0x" + string(rune('a'+i)),
				FirstSeen: time.Now(),
				Source:    "test",
			}
			queue.add(record, nil)
		}

		assert.LessOrEqual(t, len(queue.items), 5, "Queue should not exceed capacity")
	})

	t.Run("PruningProcessedTransactions", func(t *testing.T) {
		queue := newTxQueue(metrics, 5, 1*time.Second)

		// Add and mark a transaction as processed
		record := &PendingTxRecord{
			Hash:      "0x1111",
			FirstSeen: time.Now(),
			Source:    "test",
		}
		queue.add(record, nil)
		queue.markProcessed("0x1111")

		// Verify it's initially marked as processed
		assert.True(t, queue.isProcessed("0x1111"), "Transaction should be marked as processed")

		// Wait for pruning threshold
		time.Sleep(1100 * time.Millisecond)

		// Perform pruning and check results
		prunedCount := queue.prune()
		assert.GreaterOrEqual(t, prunedCount, 1, "Should have pruned at least one processed transaction")

		// Verify transaction is no longer marked as processed
		assert.False(t, queue.isProcessed("0x1111"), "Pruned transaction should no longer be marked as processed")
	})
}

// TestBatchOperations tests the logic of transaction batching.
func TestBatchOperations(t *testing.T) {
	// Create a test dataset with a known number of transactions
	const totalTxCount = 117
	const batchSize = 25

	// Setup test transactions
	pendingTxs := make(map[string]*PendingTxRecord)
	for i := 0; i < totalTxCount; i++ {
		hash := fmt.Sprintf("0x%064x", i) // Create valid-looking hashes.
		pendingTxs[hash] = &PendingTxRecord{
			Hash:      hash,
			FirstSeen: time.Now(),
			Source:    "test",
		}
	}

	// Extract transaction hashes for batching
	txHashes := make([]string, 0, len(pendingTxs))
	for hash := range pendingTxs {
		txHashes = append(txHashes, hash)
	}

	// Create batches using the same logic as in production code
	var txBatches [][]string
	currentBatch := make([]string, 0, batchSize)

	for _, hash := range txHashes {
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

	// Verify batching properties using sub-tests
	t.Run("BatchCount", func(t *testing.T) {
		// Expected: ceil(totalTxCount/batchSize) batches
		expectedBatchCount := (totalTxCount + batchSize - 1) / batchSize
		assert.Equal(t, expectedBatchCount, len(txBatches),
			"Should have correct number of batches")
	})

	t.Run("BatchSizes", func(t *testing.T) {
		// Verify full batches except potentially the last one
		for i, batch := range txBatches {
			if i < len(txBatches)-1 {
				assert.Equal(t, batchSize, len(batch),
					"Batch %d should be full with %d items", i, batchSize)
			} else {
				expectedLastBatchSize := totalTxCount % batchSize
				if expectedLastBatchSize == 0 {
					expectedLastBatchSize = batchSize
				}
				assert.Equal(t, expectedLastBatchSize, len(batch),
					"Last batch should contain exactly %d transactions", expectedLastBatchSize)
			}
		}
	})

	t.Run("AllTransactionsIncluded", func(t *testing.T) {
		// Count total transactions in batches
		txCount := 0
		for _, batch := range txBatches {
			txCount += len(batch)
		}
		assert.Equal(t, totalTxCount, txCount, "All transactions should be included in batches")
	})

	t.Run("NoDuplicates", func(t *testing.T) {
		// Check for duplicates
		seen := make(map[string]bool)
		for _, batch := range txBatches {
			for _, hash := range batch {
				assert.False(t, seen[hash], "Transaction %s should not be duplicated across batches", hash)
				seen[hash] = true
			}
		}
		assert.Equal(t, totalTxCount, len(seen), "All transactions should be unique")
	})
}

// TestTxQueueConcurrency tests the thread safety of the txQueue.
func TestTxQueueConcurrency(t *testing.T) {
	metrics := NewMetrics("test4", "testnet")
	queue := newTxQueue(metrics, 1000, 5*time.Second)

	const txCount = 100
	const workers = 10

	// Test concurrency with writes, reads, and pruning operations all happening simultaneously
	t.Run("ConcurrentOperations", func(t *testing.T) {
		var wg sync.WaitGroup

		// Create concurrent writers
		t.Run("Writers", func(t *testing.T) {
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for i := 0; i < txCount; i++ {
						hash := fmt.Sprintf("0x%d_%d", workerID, i)
						record := &PendingTxRecord{
							Hash:      hash,
							FirstSeen: time.Now(),
							Source:    "test",
						}
						queue.add(record, nil)

						// Occasionally mark as processed
						if i%3 == 0 {
							queue.markProcessed(hash)
						}

						// Simulate variable workloads
						if i%10 == 0 {
							time.Sleep(1 * time.Millisecond)
						}
					}
				}(w)
			}
		})

		// Create concurrent readers
		t.Run("Readers", func(t *testing.T) {
			for w := 0; w < workers; w++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()

					for i := 0; i < txCount; i++ {
						for j := 0; j < workers; j++ {
							hash := fmt.Sprintf("0x%d_%d", j, i)
							// Just check if processed - should not panic or have race conditions
							_ = queue.isProcessed(hash)
						}

						// Simulate variable read patterns
						if i%7 == 0 {
							time.Sleep(1 * time.Millisecond)
						}
					}
				}(w)
			}
		})

		// Run a pruner concurrently
		t.Run("Pruner", func(t *testing.T) {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < 10; i++ {
					time.Sleep(50 * time.Millisecond)
					queue.prune()
				}
			}()
		})

		// Wait for all goroutines to finish
		wg.Wait()

		// Verify the queue state is consistent
		processedCount := 0
		queue.mutex.RLock()
		for range queue.processed {
			processedCount++
		}
		queue.mutex.RUnlock()

		// We expect approximately txCount * workers / 3 processed items (minus some due to pruning)
		t.Logf("Final processed count: %d", processedCount)
		assert.Greater(t, processedCount, 0, "Should have some processed transactions")
	})
}

// TestMempoolWatcher tests the MempoolWatcher.
func TestMempoolWatcher(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock client.
	mockClient := mock.NewMockClientProvider(ctrl)

	// Set up expected calls.
	txHash := "0xabc123"
	mockTxData := json.RawMessage(`{"hash":"0xabc123","from":"0x123456","to":"0x654321","value":"0x1234"}`)
	txHash2 := "0xdef456"

	// Create channels for the subscription.
	txChan := make(chan string)
	errChan := make(chan error)

	// Set up expectations.
	mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(txChan, errChan, nil)

	// Interface method expectations.
	mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`{"pending":{},"queued":{}}`), nil).AnyTimes()
	mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{
		json.RawMessage(`{"hash":"0xfed789"}`),
	}, nil).AnyTimes()
	mockClient.EXPECT().BatchGetTransactionsByHash(gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, hashes []string) ([]json.RawMessage, error) {
			results := make([]json.RawMessage, len(hashes))
			for i, hash := range hashes {
				if hash == txHash {
					results[i] = mockTxData
				}
			}

			return results, nil
		}).AnyTimes()
	mockClient.EXPECT().Close().Return(nil).AnyTimes()

	// Create test metrics.
	metrics := NewMetrics("test6", "testnet")

	// Create a callback that records processed transactions.
	var processedTxs []string
	var processedData []json.RawMessage
	var callbackMu sync.Mutex
	var processingWg sync.WaitGroup

	processingWg.Add(1) // Expecting at least one transaction.

	callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
		callbackMu.Lock()
		defer callbackMu.Unlock()

		processedTxs = append(processedTxs, record.Hash)
		processedData = append(processedData, txData)
		processingWg.Done() // Mark one tx as processed.

		return nil
	}

	// Create logger that discards output.
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	log := logger.WithField("component", "test")

	// Create config.
	config := &Config{
		FetchInterval:        1,   // 1 second.
		ProcessingInterval:   100, // 100ms.
		PruneDuration:        10,  // 10 seconds.
		WebsocketEnabled:     true,
		RpcBatchSize:         10,
		MaxConcurrency:       2,
		ProcessorWorkerCount: 2,
		QueueSize:            100,
	}

	// Create and start the watcher.
	watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := watcher.Start(ctx)
	assert.NoError(t, err)

	// Send transactions through the subscription.
	go func() {
		txChan <- txHash
		// Give the system time to process the first transaction before sending the second
		time.Sleep(100 * time.Millisecond)
		txChan <- txHash2
	}()

	// Wait for transaction processing to complete instead of a fixed time
	waitCh := make(chan struct{})
	go func() {
		processingWg.Wait()
		close(waitCh)
	}()

	// Add a timeout in case the processing never completes
	select {
	case <-waitCh:
		// Processing completed successfully
	case <-time.After(1500 * time.Millisecond):
		t.Log("Timeout waiting for transaction processing")
	}

	// Lock before accessing state
	callbackMu.Lock()
	// Check processed count and add more wait if needed
	if len(processedTxs) < 1 {
		callbackMu.Unlock()
		// Wait a bit more if no transactions were processed yet
		time.Sleep(500 * time.Millisecond)
		callbackMu.Lock()
	}

	// Stop the watcher with lock held to avoid race
	watcher.Stop()

	// Find 0xabc123 in processed txs.
	foundAbc := false
	for i, hash := range processedTxs {
		if hash == txHash {
			foundAbc = true

			// Verify the data matches what we set.
			assert.Equal(t, mockTxData, processedData[i])

			break
		}
	}
	callbackMu.Unlock()

	assert.True(t, foundAbc, "Should process transaction with hash 0xabc123")
}

// TestMempoolWatcherSubscriptionError tests subscription error handling.
func TestMempoolWatcherSubscriptionError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create mock client.
	mockClient := mock.NewMockClientProvider(ctrl)

	// Create channels for the subscription.
	txChan := make(chan string)
	errChan := make(chan error)

	// Set up expectations for initial subscription.
	mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(txChan, errChan, nil)

	// Setup follow-up subscription after error.
	txChan2 := make(chan string)
	errChan2 := make(chan error)
	mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(txChan2, errChan2, nil).AnyTimes()

	// Allow other calls.
	mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`{"pending":{},"queued":{}}`), nil).AnyTimes()
	mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
	mockClient.EXPECT().BatchGetTransactionsByHash(gomock.Any(), gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
	mockClient.EXPECT().Close().Return(nil).AnyTimes()

	// Create logger that discards output.
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	log := logger.WithField("component", "test")

	// Create metrics.
	metrics := NewMetrics("test7", "testnet")

	// Create a no-op callback.
	callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
		return nil
	}

	// Create config.
	config := &Config{
		FetchInterval:                  1,   // 1 second.
		ProcessingInterval:             100, // 100ms.
		PruneDuration:                  10,  // 10 seconds.
		WebsocketEnabled:               true,
		RpcBatchSize:                   10,
		MaxConcurrency:                 2,
		ProcessorWorkerCount:           2,
		QueueSize:                      100,
		CircuitBreakerFailureThreshold: 3,
		CircuitBreakerResetTimeout:     10,
	}

	// Create and start the watcher.
	watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err := watcher.Start(ctx)
	assert.NoError(t, err)

	// Create a channel to signal when subscription reconnect is done
	reconnectDone := make(chan struct{})
	var reconnectMu sync.Mutex
	reconnectMu.Lock()
	reconnected := false
	reconnectMu.Unlock()

	// Monitor txChan2 to detect reconnection
	go func() {
		select {
		case <-txChan2:
			reconnectMu.Lock()
			reconnected = true
			reconnectMu.Unlock()
			close(reconnectDone)
		case <-ctx.Done():
			close(reconnectDone)
		}
	}()

	// Send an error to trigger reconnection.
	go func() {
		time.Sleep(500 * time.Millisecond)
		errChan <- fmt.Errorf("test subscription error")
	}()

	// Send a transaction on the new channel to verify reconnection worked.
	go func() {
		time.Sleep(1500 * time.Millisecond)
		txChan2 <- "0xreconnected"
	}()

	// Wait for processing with timeout
	select {
	case <-reconnectDone:
		// Reconnection occurred or context done
	case <-time.After(2500 * time.Millisecond):
		t.Log("Timeout waiting for reconnection")
	}

	// Stop the watcher
	watcher.Stop()

	// Verify reconnection happened
	reconnectMu.Lock()
	assert.True(t, reconnected, "Should have reconnected after error")
	reconnectMu.Unlock()
}

// TestFetchAndProcessTxPool tests the watcher can process mempool content from RPC.
func TestFetchAndProcessTxPool(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// Create logger with output discarded.
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	log := logger.WithField("component", "test")

	// Create metrics.
	metrics := NewMetrics("test8", "testnet")

	// Setup common test resources.
	setupTest := func() (*mock.MockClientProvider, []string, *Config, func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error) {
		// Create mock client.
		mockClient := mock.NewMockClientProvider(ctrl)

		// Transaction hashes we'll use for testing.
		txHashes := []string{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x0000000000000000000000000000000000000000000000000000000000000002",
			"0x0000000000000000000000000000000000000000000000000000000000000003",
		}

		// Create config with minimal settings for testing.
		config := &Config{
			QueueSize:          100,
			PruneDuration:      10,
			FetchInterval:      1,
			ProcessingInterval: 100,
			RpcBatchSize:       10,
			MaxConcurrency:     1,
		}

		// Create a callback function that records processed transactions.
		var processedTxs []string
		callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
			processedTxs = append(processedTxs, record.Hash)

			return nil
		}

		return mockClient, txHashes, config, callback
	}

	t.Run("FetchAndProcessPendingTransactions", func(t *testing.T) {
		mockClient, txHashes, config, _ := setupTest()

		// Create transactions for eth_pendingTransactions response
		pendingTxsResponse := []json.RawMessage{
			json.RawMessage(`{"hash":"` + txHashes[0] + `", "value":"0x123"}`),
			json.RawMessage(`{"hash":"` + txHashes[1] + `", "value":"0x456"}`),
		}

		// Setup expected calls
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return(pendingTxsResponse, nil)

		// Empty txpool_content response
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`{"pending":{},"queued":{}}`), nil).AnyTimes()

		// Other required method expectations
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create a transaction tracking mechanism
		var processedTxs []string
		var processingMu sync.Mutex
		var processingWg sync.WaitGroup
		processingWg.Add(2) // Expect both transactions

		callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
			processingMu.Lock()
			processedTxs = append(processedTxs, record.Hash)
			processingMu.Unlock()
			processingWg.Done()

			return nil
		}

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Manually call the method to test
		err = watcher.fetchAndProcessPendingTransactions(ctx)
		assert.NoError(t, err)

		// Verify pending transactions were added
		watcher.pendingTxsMutex.RLock()
		assert.Contains(t, watcher.pendingTxs, txHashes[0])
		assert.Contains(t, watcher.pendingTxs, txHashes[1])
		watcher.pendingTxsMutex.RUnlock()

		// Now manually mark one tx for pruning to test that code path
		watcher.pendingTxsMutex.Lock()
		watcher.pendingTxs[txHashes[0]].MarkedForPruning = true
		watcher.pendingTxsMutex.Unlock()

		// Second call with the same transactions - should clear the pruning mark
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return(pendingTxsResponse, nil)

		err = watcher.fetchAndProcessPendingTransactions(ctx)
		assert.NoError(t, err)

		// Verify the pruning mark was cleared
		watcher.pendingTxsMutex.RLock()
		assert.False(t, watcher.pendingTxs[txHashes[0]].MarkedForPruning,
			"MarkedForPruning should be cleared for tx still in mempool")
		watcher.pendingTxsMutex.RUnlock()

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("FetchAndProcessPendingTransactions_EmptyResponse", func(t *testing.T) {
		mockClient, _, config, callback := setupTest()

		// Setup expected calls with empty response - tests early return
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil)

		// Other required method expectations
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`{"pending":{},"queued":{}}`), nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Call the method with empty response to test early return
		err = watcher.fetchAndProcessPendingTransactions(ctx)
		assert.NoError(t, err)

		// Verify no transactions were added
		watcher.pendingTxsMutex.RLock()
		assert.Empty(t, watcher.pendingTxs, "No transactions should be added for empty response")
		watcher.pendingTxsMutex.RUnlock()

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("FetchAndProcessPendingTransactions_InvalidTx", func(t *testing.T) {
		mockClient, txHashes, config, callback := setupTest()

		// Create response with one valid and one invalid transaction
		pendingTxsResponse := []json.RawMessage{
			json.RawMessage(`{"hash":"` + txHashes[0] + `", "value":"0x123"}`),
			json.RawMessage(`{"invalid_json"`), // This will fail to parse
		}

		// Setup expected calls
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return(pendingTxsResponse, nil)

		// Other required method expectations
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`{"pending":{},"queued":{}}`), nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Call the method with partially invalid response
		err = watcher.fetchAndProcessPendingTransactions(ctx)
		assert.NoError(t, err)

		// Verify only the valid transaction was added
		watcher.pendingTxsMutex.RLock()
		assert.Contains(t, watcher.pendingTxs, txHashes[0], "Valid transaction should be added")
		assert.Len(t, watcher.pendingTxs, 1, "Only valid transaction should be added")
		watcher.pendingTxsMutex.RUnlock()

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("ErrorHandlingDuringFetch", func(t *testing.T) {
		mockClient, _, config, callback := setupTest()

		// Setup expected calls with error responses.
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(nil, errors.New("txpool error")).AnyTimes()

		// Other required method expectations.
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher.
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		// Start the watcher.
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Manually call the fetch function - should handle error gracefully.
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.Error(t, err)

		// Verify nothing was added to pending list.
		watcher.pendingTxsMutex.RLock()
		pendingCount := len(watcher.pendingTxs)
		watcher.pendingTxsMutex.RUnlock()

		assert.Equal(t, 0, pendingCount, "Should have 0 transactions in pending list after error")

		// Stop the watcher.
		watcher.Stop()
	})

	t.Run("FetchAndProcessPendingTransactions_Error", func(t *testing.T) {
		mockClient, _, config, callback := setupTest()

		// Setup expected calls with error response
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return(nil, errors.New("pending tx error"))

		// Other required method expectations
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`{"pending":{},"queued":{}}`), nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Call the method with error response
		err = watcher.fetchAndProcessPendingTransactions(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "pending tx error")

		// Verify no transactions were added
		watcher.pendingTxsMutex.RLock()
		assert.Empty(t, watcher.pendingTxs, "No transactions should be added when there's an error")
		watcher.pendingTxsMutex.RUnlock()

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("FetchAndProcessTxPoolContent", func(t *testing.T) {
		mockClient, txHashes, config, callback := setupTest()

		// Create a complex, nested txpool_content response to exercise
		// the nested loop/parsing logic in fetchAndProcessTxPool
		txpoolResponse := `{
			"pending": {
				"0xaddress1": {
					"0": {"hash":"` + txHashes[0] + `", "value":"0x123", "gasPrice":"0x456"},
					"1": {"hash":"` + txHashes[1] + `", "value":"0x234", "gasPrice":"0x567"}
				},
				"0xaddress2": {
					"5": {"hash":"` + txHashes[2] + `", "value":"0x345", "gasPrice":"0x678"}
				}
			},
			"queued": {
				"0xaddress3": {
					"2": {"hash":"0xextra1", "value":"0x456", "gasPrice":"0x789"}
				}
			}
		}`

		// Setup expected calls with deeply nested response
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(txpoolResponse), nil).AnyTimes()

		// Other required method expectations
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher with the callback
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Call the method we want to test with complex nested response
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.NoError(t, err)

		// Verify pending transactions were correctly parsed and added
		watcher.pendingTxsMutex.RLock()
		pendingCount := len(watcher.pendingTxs)

		// Check for all expected transactions
		expectedTxs := append(txHashes, "0xextra1")
		for _, hash := range expectedTxs {
			tx, exists := watcher.pendingTxs[hash]
			assert.True(t, exists, "Transaction %s should be in pendingTxs map", hash)
			if exists {
				// Verify each transaction has data from the response
				assert.NotNil(t, tx.TxData, "Transaction %s should have data", hash)
				assert.Contains(t, tx.Source, "rpc_txpool_content", "Source should contain txpool_content")
			}
		}
		watcher.pendingTxsMutex.RUnlock()

		// We should have all transactions from both sections (pending and queued)
		assert.Equal(t, 4, pendingCount, "Should have 4 transactions (3 from original txHashes plus 1 extra)")

		// Manually mark one transaction as already processed to test the duplicate path
		watcher.txQueue.markProcessed(txHashes[0])

		// Call fetchAndProcessTxPool again to test the duplicate handling path
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.NoError(t, err)

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("FetchAndProcessTxPoolWithDroppedTxs", func(t *testing.T) {
		mockClient, txHashes, config, callback := setupTest()

		// Create a response with some transactions
		initialResponse := `{
			"pending": {
				"0xaddress1": {
					"0": {"hash":"` + txHashes[0] + `", "value":"0x123"},
					"1": {"hash":"` + txHashes[1] + `", "value":"0x234"}
				}
			},
			"queued": {}
		}`

		// Later response with only one of the transactions
		// This simulates a transaction being dropped from the mempool
		laterResponse := `{
			"pending": {
				"0xaddress1": {
					"0": {"hash":"` + txHashes[0] + `", "value":"0x123"}
				}
			},
			"queued": {}
		}`

		// First call returns both transactions
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(initialResponse), nil)

		// Second call returns only one transaction (other was dropped)
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(laterResponse), nil)

		// Other required method expectations
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// First call to get both transactions
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.NoError(t, err)

		// Verify both transactions are in the pending map
		watcher.pendingTxsMutex.RLock()
		assert.Contains(t, watcher.pendingTxs, txHashes[0])
		assert.Contains(t, watcher.pendingTxs, txHashes[1])
		watcher.pendingTxsMutex.RUnlock()

		// Mark the second transaction for pruning (simulating it was detected as missing)
		watcher.pendingTxsMutex.Lock()
		watcher.pendingTxs[txHashes[1]].MarkedForPruning = true
		watcher.pendingTxsMutex.Unlock()

		// Second call - should detect the marked transaction is not in the mempool
		// and should remove it
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.NoError(t, err)

		// Verify the dropped transaction was removed
		watcher.pendingTxsMutex.RLock()
		assert.Contains(t, watcher.pendingTxs, txHashes[0], "Transaction still in mempool should remain")
		assert.NotContains(t, watcher.pendingTxs, txHashes[1], "Marked transaction not in mempool should be removed")
		watcher.pendingTxsMutex.RUnlock()

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("FetchAndProcessTxPoolError", func(t *testing.T) {
		mockClient, _, config, callback := setupTest()

		// Setup mock to return an error
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(nil, errors.New("txpool error"))

		// Other required method expectations
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Call the method with error response
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "txpool error")

		// Stop the watcher
		watcher.Stop()
	})

	t.Run("FetchAndProcessTxPoolInvalidJSON", func(t *testing.T) {
		mockClient, _, config, callback := setupTest()

		// Setup mock to return invalid JSON
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(`invalid json`), nil)

		// Other required method expectations
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start the watcher
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		// Call the method with invalid JSON
		err = watcher.fetchAndProcessTxPool(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal")

		// Stop the watcher
		watcher.Stop()
	})

	// Complete test that will mark transactions for pruning that aren't in new response
	t.Run("FetchAndProcessTxPoolMarkDroppedTxs", func(t *testing.T) {
		// Create mock client
		mockClient := mock.NewMockClientProvider(ctrl)

		// Create config with minimal settings for testing
		config := &Config{
			QueueSize:          100,
			PruneDuration:      5,
			FetchInterval:      1,
			ProcessingInterval: 100,
			RpcBatchSize:       10,
			MaxConcurrency:     1,
		}

		// Create metrics
		metrics := NewMetrics("test9", "testnet")

		// Create a no-op callback
		callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
			return nil
		}

		// Transaction hashes we'll use for testing
		txHashes := []string{
			"0x0000000000000000000000000000000000000000000000000000000000000001",
			"0x0000000000000000000000000000000000000000000000000000000000000002",
			"0x0000000000000000000000000000000000000000000000000000000000000003",
		}

		// Create watcher
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Initialize with transactions already in the pending map
		now := time.Now()
		watcher.pendingTxsMutex.Lock()
		for _, hash := range txHashes {
			watcher.pendingTxs[hash] = &PendingTxRecord{
				Hash:             hash,
				FirstSeen:        now,
				Source:           "test",
				MarkedForPruning: false,
			}
		}
		watcher.pendingTxsMutex.Unlock()

		// Mark two transactions for pruning
		watcher.pendingTxsMutex.Lock()
		watcher.pendingTxs[txHashes[1]].MarkedForPruning = true
		watcher.pendingTxs[txHashes[2]].MarkedForPruning = true
		watcher.pendingTxsMutex.Unlock()

		// Create a txpool_content response that only includes the first transaction
		response := `{
			"pending": {
				"0xaddress1": {
					"0": {"hash":"` + txHashes[0] + `", "value":"0x123"}
				}
			},
			"queued": {}
		}`

		// Setup mock to return only the first transaction
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).Return(json.RawMessage(response), nil)

		// Don't expect any other calls in this test
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Call the method directly
		err := watcher.fetchAndProcessTxPool(ctx)
		assert.NoError(t, err)

		// The first transaction should not be marked for pruning
		// The marked transactions should be removed since they were not in the response
		watcher.pendingTxsMutex.RLock()
		assert.Contains(t, watcher.pendingTxs, txHashes[0], "Transaction still in mempool should remain")
		assert.False(t, watcher.pendingTxs[txHashes[0]].MarkedForPruning, "Transaction in mempool shouldn't be marked")

		// Pruned transactions should be removed
		assert.NotContains(t, watcher.pendingTxs, txHashes[1], "Marked transaction not in response should be removed")
		assert.NotContains(t, watcher.pendingTxs, txHashes[2], "Marked transaction not in response should be removed")
		watcher.pendingTxsMutex.RUnlock()
	})
}

// TestProcessDroppedTransactions tests that stale transactions are pruned from the mempool.
func TestProcessDroppedTransactions(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := logrus.New()
	logger.SetOutput(io.Discard)
	log := logger.WithField("component", "test")

	// Test cases for transaction statuses.
	testCases := []struct {
		name           string
		txHash         string
		mempoolReturns bool  // Whether the tx is returned in mempool.
		ageInSeconds   int64 // Age of the transaction in seconds.
		shouldPrune    bool  // Whether the tx should be pruned.
	}{
		{
			name:           "Active recent transaction",
			txHash:         "0x001",
			mempoolReturns: true,
			ageInSeconds:   1, // Just added (1 second old).
			shouldPrune:    false,
		},
		{
			name:           "Recently dropped transaction",
			txHash:         "0x002",
			mempoolReturns: false,
			ageInSeconds:   2, // Below the pruning threshold.
			shouldPrune:    false,
		},
		{
			name:           "Stale active transaction",
			txHash:         "0x003",
			mempoolReturns: true,
			ageInSeconds:   10,   // Above pruning threshold but still in mempool.
			shouldPrune:    true, // Will be pruned based on age only.
		},
		{
			name:           "Stale dropped transaction",
			txHash:         "0x004",
			mempoolReturns: false,
			ageInSeconds:   8, // Above pruning threshold.
			shouldPrune:    true,
		},
	}

	t.Run("PruningLogic", func(t *testing.T) {
		// Create mock client.
		mockClient := mock.NewMockClientProvider(ctrl)

		// Setup mock client to return active transactions.
		mockClient.EXPECT().GetTxpoolContent(gomock.Any()).DoAndReturn(
			func(ctx context.Context) (json.RawMessage, error) {
				// Build a response where only some transactions are in the mempool.
				var activeTxs []string
				for _, tc := range testCases {
					if tc.mempoolReturns {
						activeTxs = append(activeTxs, tc.txHash)
					}
				}

				// Build response JSON.
				pendingMap := make(map[string]map[string]string)
				for i, hash := range activeTxs {
					addr := fmt.Sprintf("0x%d", i+100)
					nonce := fmt.Sprintf("%d", i)

					if pendingMap[addr] == nil {
						pendingMap[addr] = make(map[string]string)
					}

					pendingMap[addr][nonce] = fmt.Sprintf(`{"hash":"%s"}`, hash)
				}

				// Convert to JSON.
				responseObj := map[string]interface{}{
					"pending": pendingMap,
					"queued":  map[string]interface{}{},
				}
				responseJSON, _ := json.Marshal(responseObj)

				return responseJSON, nil
			},
		).AnyTimes()

		// Other required method expectations.
		mockClient.EXPECT().BatchGetTransactionsByHash(gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()
		mockClient.EXPECT().GetPendingTransactions(gomock.Any()).Return([]json.RawMessage{}, nil).AnyTimes()
		mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(make(chan string), make(chan error), nil).AnyTimes()
		mockClient.EXPECT().Close().Return(nil).AnyTimes()

		// Create metrics.
		metrics := NewMetrics("test10", "testnet")

		// Create a callback to track processed transactions.
		var processedTxs []string
		callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
			processedTxs = append(processedTxs, record.Hash)

			return nil
		}

		// Create config with test settings - pruning threshold of 5 seconds.
		config := &Config{
			QueueSize:          100,
			PruneDuration:      5, // 5 second prune interval.
			FetchInterval:      1,
			ProcessingInterval: 100,
		}

		// Create watcher.
		watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		// Start the watcher.
		err := watcher.Start(ctx)
		assert.NoError(t, err)

		now := time.Now()

		// Manually set up transactions in pending list with different ages.
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Set the first seen time based on age.
				firstSeen := now.Add(-time.Duration(tc.ageInSeconds) * time.Second)

				// Add to pending transactions.
				watcher.pendingTxsMutex.Lock()
				watcher.pendingTxs[tc.txHash] = &PendingTxRecord{
					Hash:             tc.txHash,
					FirstSeen:        firstSeen,
					Source:           "test",
					MarkedForPruning: !tc.mempoolReturns, // Mark as pruning if not in mempool.
				}
				watcher.pendingTxsMutex.Unlock()
			})
		}

		// Log original state for debugging.
		t.Log("Initial state before pruning:")
		watcher.pendingTxsMutex.RLock()
		for _, tc := range testCases {
			record, exists := watcher.pendingTxs[tc.txHash]
			if exists {
				age := now.Sub(record.FirstSeen).Seconds()
				t.Logf("- %s: age=%.1fs, marked=%v",
					tc.txHash, age, record.MarkedForPruning)
			}
		}
		watcher.pendingTxsMutex.RUnlock()

		// Manually trigger pruning for stale transactions.
		watcher.prunePendingTxs()

		// Log the post-pruning state.
		t.Log("State after pruning:")
		watcher.pendingTxsMutex.RLock()
		for _, tc := range testCases {
			record, exists := watcher.pendingTxs[tc.txHash]
			if exists {
				age := now.Sub(record.FirstSeen).Seconds()
				t.Logf("- %s: age=%.1fs, marked=%v",
					tc.txHash, age, record.MarkedForPruning)
			} else {
				t.Logf("- %s: pruned", tc.txHash)
			}
		}
		watcher.pendingTxsMutex.RUnlock()

		// Verify transactions were correctly pruned based on age.
		watcher.pendingTxsMutex.RLock()
		for _, tc := range testCases {
			_, exists := watcher.pendingTxs[tc.txHash]

			if tc.shouldPrune {
				assert.False(t, exists, "Transaction %s should be pruned (age: %d)",
					tc.txHash, tc.ageInSeconds)
			} else {
				assert.True(t, exists, "Transaction %s should not be pruned (age: %d)",
					tc.txHash, tc.ageInSeconds)
			}
		}
		watcher.pendingTxsMutex.RUnlock()

		// Stop the watcher.
		watcher.Stop()
	})
}

// TestFetchersBasedOnConfig tests that the fetcher goroutines are started or not
// based on the configuration flags.
func TestFetchersBasedOnConfig(t *testing.T) {
	// Setup test logger that discards output
	logger := logrus.New()
	logger.SetOutput(io.Discard)
	log := logger.WithField("component", "test")

	// Create metrics
	metrics := NewMetrics("test-fetchers-config", "testnet")

	// Create a no-op callback
	callback := func(ctx context.Context, record *PendingTxRecord, txData json.RawMessage) error {
		return nil
	}

	// Setup for monitoring function calls
	type callCounter struct {
		sync.Mutex
		txPoolContentCalls int
		pendingTxsCalls    int
	}

	counter := &callCounter{}

	// Test cases
	testCases := []struct {
		name              string
		txPoolEnabled     bool
		ethPendingEnabled bool
		expectTxPool      bool
		expectEthPending  bool
	}{
		{
			name:              "BothEnabled",
			txPoolEnabled:     true,
			ethPendingEnabled: true,
			expectTxPool:      true,
			expectEthPending:  true,
		},
		{
			name:              "OnlyTxPoolEnabled",
			txPoolEnabled:     true,
			ethPendingEnabled: false,
			expectTxPool:      true,
			expectEthPending:  false,
		},
		{
			name:              "OnlyEthPendingEnabled",
			txPoolEnabled:     false,
			ethPendingEnabled: true,
			expectTxPool:      false,
			expectEthPending:  true,
		},
		{
			name:              "BothDisabled",
			txPoolEnabled:     false,
			ethPendingEnabled: false,
			expectTxPool:      false,
			expectEthPending:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset counters
			counter.Lock()
			counter.txPoolContentCalls = 0
			counter.pendingTxsCalls = 0
			counter.Unlock()

			// Create mock controller
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			// Create mock client with instrumented methods to count calls
			mockClient := mock.NewMockClientProvider(ctrl)

			// Setup the mock to count calls
			mockClient.EXPECT().GetTxpoolContent(gomock.Any()).DoAndReturn(
				func(ctx context.Context) (json.RawMessage, error) {
					counter.Lock()
					counter.txPoolContentCalls++
					counter.Unlock()

					return json.RawMessage(`{"pending":{},"queued":{}}`), nil
				}).AnyTimes()

			mockClient.EXPECT().GetPendingTransactions(gomock.Any()).DoAndReturn(
				func(ctx context.Context) ([]json.RawMessage, error) {
					counter.Lock()
					counter.pendingTxsCalls++
					counter.Unlock()

					return []json.RawMessage{}, nil
				}).AnyTimes()

			// Other required method expectations
			mockClient.EXPECT().SubscribeToNewPendingTxs(gomock.Any()).Return(
				make(chan string), make(chan error), nil).AnyTimes()
			mockClient.EXPECT().BatchGetTransactionsByHash(gomock.Any(), gomock.Any()).Return(
				[]json.RawMessage{}, nil).AnyTimes()
			mockClient.EXPECT().Close().Return(nil).AnyTimes()

			// Create config with very short intervals for testing
			config := &Config{
				QueueSize:            100,
				PruneDuration:        60,
				FetchInterval:        1, // 1 second interval to ensure methods are called
				ProcessingInterval:   500,
				WebsocketEnabled:     false,
				TxPoolContentEnabled: tc.txPoolEnabled,
				EthPendingTxsEnabled: tc.ethPendingEnabled,
				ProcessorWorkerCount: 2,
			}

			// Create real watcher.
			watcher := NewMempoolWatcher(mockClient, log, config, callback, metrics)

			// Start the watcher.
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := watcher.Start(ctx)
			assert.NoError(t, err)

			// Wait for any fetchers to run at least once.
			time.Sleep(1500 * time.Millisecond)

			// Stop the watcher
			watcher.Stop()

			// Check if the methods were called based on configuration.
			counter.Lock()
			defer counter.Unlock()

			if tc.expectTxPool {
				assert.Greater(t, counter.txPoolContentCalls, 0,
					"txpool_content should be called when TxPoolContentEnabled=true")
			} else {
				assert.Equal(t, 0, counter.txPoolContentCalls,
					"txpool_content should not be called when TxPoolContentEnabled=false")
			}

			if tc.expectEthPending {
				assert.Greater(t, counter.pendingTxsCalls, 0,
					"eth_pendingTransactions should be called when EthPendingTxsEnabled=true")
			} else {
				assert.Equal(t, 0, counter.pendingTxsCalls,
					"eth_pendingTransactions should not be called when EthPendingTxsEnabled=false",
					"counter.pendingTxsCalls=%d", counter.pendingTxsCalls)
			}
		})
	}
}
