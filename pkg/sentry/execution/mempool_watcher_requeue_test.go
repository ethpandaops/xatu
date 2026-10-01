package execution

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// unusedClientProvider is a ClientProvider whose methods are never expected
// to be called by this test: WebsocketEnabled is false and
// addPendingTransaction is driven directly.
type unusedClientProvider struct{}

func (unusedClientProvider) GetTxpoolContent(_ context.Context) (json.RawMessage, error) {
	return nil, errors.New("not used in this test")
}

func (unusedClientProvider) GetPendingTransactions(_ context.Context) ([]json.RawMessage, error) {
	return nil, errors.New("not used in this test")
}

func (unusedClientProvider) BatchGetTransactionsByHash(_ context.Context, _ []string) ([]json.RawMessage, error) {
	return nil, errors.New("not used in this test")
}

func (unusedClientProvider) CallContext(_ context.Context, _ any, _ string, _ ...any) error {
	return errors.New("not used in this test")
}

func (unusedClientProvider) SubscribeToNewPendingTxs(_ context.Context) (<-chan string, <-chan error, error) {
	return nil, nil, errors.New("not used in this test")
}

// A transaction seen hash-only (e.g. over the WebSocket subscription) that
// later gains its data from a second source (e.g. a txpool_content poll)
// used to never be requeued for processing: the guard governing that had
// already been made permanently false by an earlier line in the same
// function. It should now be requeued and processed.
func TestMempoolWatcher_TransactionThatGainsDataIsProcessed(t *testing.T) {
	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	var (
		mu        sync.Mutex
		processed []string
	)

	cfg := &Config{
		WebsocketEnabled:               false,
		TxPoolContentEnabled:           false,
		EthPendingTxsEnabled:           false,
		FetchInterval:                  15,
		PruneDuration:                  300,
		ProcessorWorkerCount:           2, // -> 1 direct worker (ProcessorWorkerCount/2)
		RpcBatchSize:                   40,
		QueueSize:                      10,
		ProcessingInterval:             500,
		MaxConcurrency:                 1,
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerResetTimeout:     30,
	}

	metrics := NewMetrics("mempool_watcher_requeue_test", "mainnet")

	w := NewMempoolWatcher(unusedClientProvider{}, log, cfg, func(_ context.Context, record *PendingTxRecord, _ json.RawMessage) error {
		mu.Lock()
		defer mu.Unlock()

		processed = append(processed, record.Hash)

		return nil
	}, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer w.Stop()

	const txHash = "0xdeadbeef00000000000000000000000000000000000000000000000000000000"

	// Hash-only sighting, as the WebSocket consumer goroutine delivers it.
	w.addPendingTransaction(txHash, nil, "ws_newPendingTransactions")

	// Give the direct worker time to drain the queue. Since the item has no
	// data yet, it goes back into pendingTxs rather than being processed,
	// waiting for a poller to enrich it.
	time.Sleep(200 * time.Millisecond)

	w.pendingTxsMutex.RLock()
	_, exists := w.pendingTxs[txHash]
	w.pendingTxsMutex.RUnlock()

	if !exists {
		t.Fatalf("expected tx to be present in pendingTxs after the hash-only sighting")
	}

	// A poller delivers the same hash with data.
	realTxData := json.RawMessage(`{"hash":"` + txHash + `","gas":"0x5208"}`)
	w.addPendingTransaction(txHash, realTxData, "txpool_content")

	deadline := time.After(2 * time.Second)
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

waitLoop:
	for {
		select {
		case <-deadline:
			break waitLoop
		case <-ticker.C:
			mu.Lock()
			done := len(processed) > 0
			mu.Unlock()

			if done {
				break waitLoop
			}
		}
	}

	mu.Lock()
	gotProcessed := append([]string(nil), processed...)
	mu.Unlock()

	if len(gotProcessed) != 1 || gotProcessed[0] != txHash {
		t.Fatalf("expected processTxCallback to be called once with %s, got %v", txHash, gotProcessed)
	}
}
