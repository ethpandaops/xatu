package execution

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

// fakeSubscribeClient is a minimal ClientProvider whose only interesting
// behavior is SubscribeToNewPendingTxs: it succeeds, then shortly after
// pushes an error onto the returned error channel to simulate a real-world
// WebSocket drop (execution client restart, load balancer reset, network
// blip). Every other ClientProvider method is unused by this test.
type fakeSubscribeClient struct {
	subscribeCalls atomic.Int32
}

func (f *fakeSubscribeClient) GetTxpoolContent(_ context.Context) (json.RawMessage, error) {
	return nil, errors.New("not used in this test")
}

func (f *fakeSubscribeClient) GetPendingTransactions(_ context.Context) ([]json.RawMessage, error) {
	return nil, errors.New("not used in this test")
}

func (f *fakeSubscribeClient) BatchGetTransactionsByHash(_ context.Context, _ []string) ([]json.RawMessage, error) {
	return nil, errors.New("not used in this test")
}

func (f *fakeSubscribeClient) CallContext(_ context.Context, _ any, _ string, _ ...any) error {
	return errors.New("not used in this test")
}

func (f *fakeSubscribeClient) SubscribeToNewPendingTxs(_ context.Context) (<-chan string, <-chan error, error) {
	f.subscribeCalls.Add(1)

	txCh := make(chan string)
	errCh := make(chan error, 1)

	go func() {
		time.Sleep(100 * time.Millisecond)
		errCh <- errors.New("simulated websocket drop (connection reset)")
	}()

	return txCh, errCh, nil
}

// The retry wrapper around the WebSocket subscription used to wait on a
// context unrelated to the subscription itself, so it never observed a
// drop and never actually reconnected. It should now call
// SubscribeToNewPendingTxs again shortly after the connection drops.
func TestMempoolWatcher_WebsocketReconnectsAfterDrop(t *testing.T) {
	client := &fakeSubscribeClient{}

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	cfg := &Config{
		WebsocketEnabled:               true,
		TxPoolContentEnabled:           false,
		EthPendingTxsEnabled:           false,
		FetchInterval:                  15,
		PruneDuration:                  300,
		ProcessorWorkerCount:           1,
		RpcBatchSize:                   40,
		QueueSize:                      10,
		ProcessingInterval:             500,
		MaxConcurrency:                 1,
		CircuitBreakerFailureThreshold: 5,
		CircuitBreakerResetTimeout:     30,
	}

	metrics := NewMetrics("mempool_watcher_reconnect_test", "mainnet")

	w := NewMempoolWatcher(client, log, cfg, func(context.Context, *PendingTxRecord, json.RawMessage) error {
		return nil
	}, metrics)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	defer w.Stop()

	// The fake client drops the subscription after 100ms. The backoff's own
	// MaxInterval is capped at 1 minute and the first retry fires almost
	// immediately, so a reconnect attempt should be well within 3 seconds.
	deadline := time.After(3 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

waitLoop:
	for {
		select {
		case <-deadline:
			break waitLoop
		case <-ticker.C:
			if client.subscribeCalls.Load() >= 2 {
				break waitLoop
			}
		}
	}

	if calls := client.subscribeCalls.Load(); calls < 2 {
		t.Fatalf("expected at least 2 SubscribeToNewPendingTxs calls within 3s of the simulated drop, got %d", calls)
	}
}
