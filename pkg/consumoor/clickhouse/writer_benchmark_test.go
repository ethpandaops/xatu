package clickhouse

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ClickHouse/ch-go"
	"github.com/ethpandaops/xatu/pkg/consumoor/route"
	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/route/all"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const benchTable = "beacon_api_eth_v1_events_head"

// benchHeadEvent returns a realistic DecoratedEvent for the head route.
func benchHeadEvent() *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
			DateTime: timestamppb.Now(),
			Id:       "bench-event-id",
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name:           "bench-client",
				Version:        "0.1.0",
				Id:             "client-bench",
				Implementation: "xatu",
				Os:             "linux",
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{
						Name: "mainnet",
						Id:   1,
					},
					Consensus: &xatu.ClientMeta_Ethereum_Consensus{
						Implementation: "lighthouse",
						Version:        "v4.5.6",
					},
				},
				AdditionalData: &xatu.ClientMeta_EthV1EventsHeadV2{
					EthV1EventsHeadV2: &xatu.ClientMeta_AdditionalEthV1EventsHeadV2Data{
						Epoch: &xatu.EpochV2{
							Number:        wrapperspb.UInt64(312_500),
							StartDateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
						},
						Slot: &xatu.SlotV2{
							Number:        wrapperspb.UInt64(10_000_000),
							StartDateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
						},
						Propagation: &xatu.PropagationV2{
							SlotStartDiff: wrapperspb.UInt64(450),
						},
					},
				},
			},
		},
		Data: &xatu.DecoratedEvent_EthV1EventsHeadV2{
			EthV1EventsHeadV2: &ethv1.EventHeadV2{
				Slot:                      wrapperspb.UInt64(10_000_000),
				Block:                     "0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
				State:                     "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
				EpochTransition:           false,
				PreviousDutyDependentRoot: "0x1111111111111111111111111111111111111111111111111111111111111111",
				CurrentDutyDependentRoot:  "0x2222222222222222222222222222222222222222222222222222222222222222",
			},
		},
	}
}

// benchWriter creates a ChGoWriter with a noop pool.Do for benchmarking.
// The returned cancel function must be called to stop background goroutines.
func benchWriter(tb testing.TB, bufferSize int) (*ChGoWriter, context.CancelFunc) {
	tb.Helper()

	ns := fmt.Sprintf("xatu_bench_%d", time.Now().UnixNano())
	metrics := telemetry.NewMetrics(ns)

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	w := &ChGoWriter{
		log:     log.WithField("component", "bench"),
		metrics: metrics,
		config: &Config{
			DrainTimeout: 30 * time.Second,
			Defaults: TableConfig{
				BufferSize: bufferSize,
			},
		},
		chgoCfg: ChGoConfig{
			MaxRetries:     0,
			RetryBaseDelay: 100 * time.Millisecond,
			RetryMaxDelay:  time.Second,
		},
		tables:         make(map[string]*chTableWriter, 4),
		batchFactories: make(map[string]func() route.ColumnarBatch, 4),
		done:           make(chan struct{}),
		poolDoFn: func(_ context.Context, _ ch.Query) error {
			return nil
		},
	}

	routes, err := tabledefs.All()
	require.NoError(tb, err)

	w.RegisterBatchFactories(routes)

	cancel := func() {
		w.stopOnce.Do(func() {
			close(w.done)
		})
	}

	return w, cancel
}

// BenchmarkWriteThroughput measures events/sec through the full
// Write() -> accumulator -> worker -> noop-flush path.
func BenchmarkWriteThroughput(b *testing.B) {
	const bufferSize = 50_000

	w, cancel := benchWriter(b, bufferSize)
	defer cancel()

	event := benchHeadEvent()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Write(benchTable, event)
	}

	b.StopTimer()

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	if err := w.FlushTables(ctx, []string{benchTable}); err != nil {
		b.Logf("FlushTables: %v", err)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkWriteConcurrent measures channel contention when multiple
// goroutines write to the same ChGoWriter concurrently.
func BenchmarkWriteConcurrent(b *testing.B) {
	const bufferSize = 100_000

	w, cancel := benchWriter(b, bufferSize)
	defer cancel()

	event := benchHeadEvent()

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			w.Write(benchTable, event)
		}
	})

	b.StopTimer()

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	if err := w.FlushTables(ctx, []string{benchTable}); err != nil {
		b.Logf("FlushTables: %v", err)
	}

	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkAccumulatorBatching measures how fast the table writer's
// run loop drains the buffer channel and builds batches via flush.
func BenchmarkAccumulatorBatching(b *testing.B) {
	const bufferSize = 50_000

	w, cancel := benchWriter(b, bufferSize)
	defer cancel()

	event := benchHeadEvent()

	// Pre-populate the buffer to measure drain/flush speed.
	// First trigger table writer creation by writing one event.
	w.Write(benchTable, event)

	// Wait briefly for the table writer goroutine to start.
	time.Sleep(10 * time.Millisecond)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		w.Write(benchTable, event)
	}

	b.StopTimer()

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	if err := w.FlushTables(ctx, []string{benchTable}); err != nil {
		b.Logf("FlushTables: %v", err)
	}

	// Count total flushes by reading the metrics.
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkEndToEndWithFlatten measures the full cost of processing one
// event from proto through FlattenTo and the Write path to batch-ready.
func BenchmarkEndToEndWithFlatten(b *testing.B) {
	const bufferSize = 50_000

	var flushCount atomic.Int64

	ns := fmt.Sprintf("xatu_bench_e2e_%d", time.Now().UnixNano())
	metrics := telemetry.NewMetrics(ns)

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	w := &ChGoWriter{
		log:     log.WithField("component", "bench_e2e"),
		metrics: metrics,
		config: &Config{
			DrainTimeout: 30 * time.Second,
			Defaults: TableConfig{
				BufferSize: bufferSize,
			},
		},
		chgoCfg: ChGoConfig{
			MaxRetries:     0,
			RetryBaseDelay: 100 * time.Millisecond,
			RetryMaxDelay:  time.Second,
		},
		tables:         make(map[string]*chTableWriter, 4),
		batchFactories: make(map[string]func() route.ColumnarBatch, 4),
		done:           make(chan struct{}),
		poolDoFn: func(_ context.Context, _ ch.Query) error {
			flushCount.Add(1)

			return nil
		},
	}

	routes, err := tabledefs.All()
	require.NoError(b, err)

	w.RegisterBatchFactories(routes)

	var wg sync.WaitGroup

	event := benchHeadEvent()

	b.ReportAllocs()
	b.ResetTimer()

	wg.Add(1)

	go func() {
		defer wg.Done()

		for i := 0; i < b.N; i++ {
			w.Write(benchTable, event)
		}
	}()

	wg.Wait()

	b.StopTimer()

	ctx, ctxCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer ctxCancel()

	if err := w.FlushTables(ctx, []string{benchTable}); err != nil {
		b.Logf("FlushTables: %v", err)
	}

	// Signal shutdown.
	w.stopOnce.Do(func() {
		close(w.done)
	})

	elapsed := b.Elapsed().Seconds()
	b.ReportMetric(float64(b.N)/elapsed, "events/sec")
	b.ReportMetric(float64(flushCount.Load()), "flushes")
	b.ReportMetric(float64(b.N)/float64(max(flushCount.Load(), 1)), "events/flush")
}

// BenchmarkConcurrentFlush measures throughput when multiple goroutines
// write and flush concurrently, exercising the concurrent INSERT path.
func BenchmarkConcurrentFlush(b *testing.B) {
	const (
		bufferSize = 100_000
		flushers   = 8
		batchSize  = 500
	)

	w, cancel := benchWriter(b, bufferSize)
	defer cancel()

	event := benchHeadEvent()

	// Pre-create the table writer.
	w.Write(benchTable, event)

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			// Write a small batch then flush, simulating concurrent WriteBatch calls.
			for j := 0; j < batchSize; j++ {
				w.Write(benchTable, event)
			}

			if err := w.FlushTables(context.Background(), []string{benchTable}); err != nil {
				b.Logf("FlushTables: %v", err)
			}
		}
	})

	b.StopTimer()
	b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds(), "events/sec")
}
