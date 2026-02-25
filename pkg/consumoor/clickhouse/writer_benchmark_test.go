package clickhouse

import (
	"context"
	"fmt"
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
func benchWriter(tb testing.TB) *ChGoWriter {
	tb.Helper()

	ns := fmt.Sprintf("xatu_bench_%d", time.Now().UnixNano())
	metrics := telemetry.NewMetrics(ns)

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	w := &ChGoWriter{
		log:     log.WithField("component", "bench"),
		metrics: metrics,
		config:  &Config{},
		chgoCfg: ChGoConfig{
			MaxRetries:     0,
			RetryBaseDelay: 100 * time.Millisecond,
			RetryMaxDelay:  time.Second,
		},
		tables:         make(map[string]*chTableWriter, 4),
		batchFactories: make(map[string]func() route.ColumnarBatch, 4),
		poolDoFn: func(_ context.Context, _ ch.Query) error {
			return nil
		},
	}

	routes, err := tabledefs.All()
	require.NoError(tb, err)

	w.RegisterBatchFactories(routes)

	return w
}

// BenchmarkFlushTableEvents measures throughput of the FlushTableEvents path
// (FlattenTo + columnar batch build + noop INSERT).
func BenchmarkFlushTableEvents(b *testing.B) {
	w := benchWriter(b)
	event := benchHeadEvent()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tableEvents := map[string][]*xatu.DecoratedEvent{
			benchTable: {event},
		}

		if err := w.FlushTableEvents(context.Background(), tableEvents); err != nil {
			b.Fatalf("FlushTableEvents: %v", err)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkFlushTableEventsBatch measures throughput with larger batches.
func BenchmarkFlushTableEventsBatch(b *testing.B) {
	const batchSize = 500

	w := benchWriter(b)
	event := benchHeadEvent()

	// Pre-build the batch of events.
	events := make([]*xatu.DecoratedEvent, batchSize)
	for i := range events {
		events[i] = event
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tableEvents := map[string][]*xatu.DecoratedEvent{
			benchTable: events,
		}

		if err := w.FlushTableEvents(context.Background(), tableEvents); err != nil {
			b.Fatalf("FlushTableEvents: %v", err)
		}
	}

	b.StopTimer()
	b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkFlushConcurrent measures throughput when multiple goroutines
// flush concurrently, exercising the concurrent INSERT path.
func BenchmarkFlushConcurrent(b *testing.B) {
	const batchSize = 500

	w := benchWriter(b)
	event := benchHeadEvent()

	events := make([]*xatu.DecoratedEvent, batchSize)
	for i := range events {
		events[i] = event
	}

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tableEvents := map[string][]*xatu.DecoratedEvent{
				benchTable: events,
			}

			if err := w.FlushTableEvents(context.Background(), tableEvents); err != nil {
				b.Logf("FlushTableEvents: %v", err)
			}
		}
	})

	b.StopTimer()
	b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds(), "events/sec")
}

// BenchmarkEndToEndWithFlatten measures the full cost of processing one
// event from proto through FlattenTo to batch-ready INSERT.
func BenchmarkEndToEndWithFlatten(b *testing.B) {
	var flushCount atomic.Int64

	ns := fmt.Sprintf("xatu_bench_e2e_%d", time.Now().UnixNano())
	metrics := telemetry.NewMetrics(ns)

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	w := &ChGoWriter{
		log:     log.WithField("component", "bench_e2e"),
		metrics: metrics,
		config:  &Config{},
		chgoCfg: ChGoConfig{
			MaxRetries:     0,
			RetryBaseDelay: 100 * time.Millisecond,
			RetryMaxDelay:  time.Second,
		},
		tables:         make(map[string]*chTableWriter, 4),
		batchFactories: make(map[string]func() route.ColumnarBatch, 4),
		poolDoFn: func(_ context.Context, _ ch.Query) error {
			flushCount.Add(1)

			return nil
		},
	}

	routes, err := tabledefs.All()
	require.NoError(b, err)

	w.RegisterBatchFactories(routes)

	event := benchHeadEvent()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		tableEvents := map[string][]*xatu.DecoratedEvent{
			benchTable: {event},
		}

		if err := w.FlushTableEvents(context.Background(), tableEvents); err != nil {
			b.Fatalf("FlushTableEvents: %v", err)
		}
	}

	b.StopTimer()

	elapsed := b.Elapsed().Seconds()
	b.ReportMetric(float64(b.N)/elapsed, "events/sec")
	b.ReportMetric(float64(flushCount.Load()), "flushes")
	b.ReportMetric(float64(b.N)/float64(max(flushCount.Load(), 1)), "events/flush")
}
