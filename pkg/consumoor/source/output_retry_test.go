package source

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	ch "github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethpandaops/xatu/pkg/clickhouse"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	retryTableA     = "table_a"
	retryTableB     = "table_b"
	retryEncoding   = "json"
	chExceptionName = "DB::Exception"
)

var errTransientWrite = errors.New("clickhouse unavailable")

// flakyWriter is a goroutine-safe Writer whose per-table failures are scripted.
type flakyWriter struct {
	mu         sync.Mutex
	failing    bool
	queued     map[string][]error
	invalidIDs map[string]struct{}
	calls      [][]string
	written    map[string][]string
}

func (w *flakyWriter) Start(context.Context) error { return nil }

func (w *flakyWriter) Stop(context.Context) error { return nil }

func (w *flakyWriter) Ping(context.Context) error { return nil }

func (w *flakyWriter) FlushTableEvents(
	_ context.Context,
	tableEvents map[string][]*xatu.DecoratedEvent,
) *clickhouse.FlushResult {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.calls = append(w.calls, slices.Sorted(maps.Keys(tableEvents)))

	if w.written == nil {
		w.written = make(map[string][]string, len(tableEvents))
	}

	result := &clickhouse.FlushResult{}

	for table, events := range tableEvents {
		if err := w.nextErr(table); err != nil {
			if result.TableErrors == nil {
				result.TableErrors = make(map[string]error, len(tableEvents))
			}

			result.TableErrors[table] = err

			continue
		}

		for _, ev := range events {
			id := ev.GetEvent().GetId()
			if _, invalid := w.invalidIDs[id]; invalid {
				result.InvalidEvents = append(result.InvalidEvents, ev)

				continue
			}

			w.written[table] = append(w.written[table], id)
		}
	}

	return result
}

func (w *flakyWriter) nextErr(table string) error {
	if w.failing {
		return errTransientWrite
	}

	errs := w.queued[table]
	if len(errs) == 0 {
		return nil
	}

	w.queued[table] = errs[1:]

	return errs[0]
}

func (w *flakyWriter) setFailing(failing bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.failing = failing
}

func (w *flakyWriter) flushCallsFor(table string) int {
	w.mu.Lock()
	defer w.mu.Unlock()

	n := 0

	for _, tables := range w.calls {
		if slices.Contains(tables, table) {
			n++
		}
	}

	return n
}

func (w *flakyWriter) writtenTo(table string) []string {
	w.mu.Lock()
	defer w.mu.Unlock()

	return slices.Clone(w.written[table])
}

func newRetryTestOutput(t *testing.T, writer Writer, rejectSink rejectSink) *xatuClickHouseOutput {
	t.Helper()

	log := logrus.New()
	log.SetLevel(logrus.ErrorLevel)

	return &xatuClickHouseOutput{
		log:      log,
		encoding: retryEncoding,
		router: newRouter(t, []route.Route{
			testRoute{eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, table: retryTableA},
			testRoute{eventName: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, table: retryTableB},
		}),
		writer:              writer,
		metrics:             newTestMetrics(),
		logSampler:          telemetry.NewLogSampler(time.Minute),
		rejectSink:          rejectSink,
		groupRetryBaseDelay: time.Millisecond,
		groupRetryMaxDelay:  4 * time.Millisecond,
	}
}

func TestWriteBatchRetriesTransientFailureInPlace(t *testing.T) {
	writer := &flakyWriter{
		queued: map[string][]error{
			retryTableB: {errTransientWrite, errTransientWrite, errTransientWrite, errTransientWrite},
		},
	}
	output := newRetryTestOutput(t, writer, &testRejectSink{})

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
		newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK), "t", 0, 2),
		newKafkaMessage(mustEventJSON(t, "e3", xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK), "t", 0, 3),
	}

	require.NoError(t, output.WriteBatch(context.Background(), msgs),
		"transient failures must be retried in place, not nacked")

	assert.Equal(t, []string{"e1"}, writer.writtenTo(retryTableA))
	assert.Equal(t, []string{"e2", "e3"}, writer.writtenTo(retryTableB))
	assert.Equal(t, 1, writer.flushCallsFor(retryTableA), "succeeded table must not be retried")
	assert.Equal(t, 5, writer.flushCallsFor(retryTableB), "failed table retried until it succeeds")
}

func TestWriteBatchRetryKeepsInvalidEventsInDLQ(t *testing.T) {
	writer := &flakyWriter{
		queued:     map[string][]error{retryTableA: {errTransientWrite}},
		invalidIDs: map[string]struct{}{"e2": {}},
	}
	rejectSink := &testRejectSink{}
	output := newRetryTestOutput(t, writer, rejectSink)

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
		newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
	}

	require.NoError(t, output.WriteBatch(context.Background(), msgs))
	assert.Equal(t, []string{"e1"}, writer.writtenTo(retryTableA))
	require.Len(t, rejectSink.records, 1)
	assert.Equal(t, rejectReasonInvalidEvent, rejectSink.records[0].Reason)
}

func TestWriteBatchRetryStopsWhenFailureTurnsPermanent(t *testing.T) {
	permanent := &ch.Exception{Code: proto.ErrCannotParseNumber, Name: chExceptionName, Message: "cannot parse number"}
	writer := &flakyWriter{
		queued: map[string][]error{retryTableA: {errTransientWrite, permanent}},
	}
	rejectSink := &testRejectSink{}
	output := newRetryTestOutput(t, writer, rejectSink)

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
		newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
	}

	require.NoError(t, output.WriteBatch(context.Background(), msgs),
		"permanent failures go to the reject sink, not back to Kafka")
	assert.Equal(t, 2, writer.flushCallsFor(retryTableA))
	require.Len(t, rejectSink.records, 2)

	for _, rec := range rejectSink.records {
		assert.Equal(t, rejectReasonWritePermanent, rec.Reason)
	}
}

func TestWriteBatchRetryNacksOnContextCancel(t *testing.T) {
	writer := &flakyWriter{failing: true}
	output := newRetryTestOutput(t, writer, &testRejectSink{})

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
		newKafkaMessage(mustEventJSON(t, "e2", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 2),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := output.WriteBatch(ctx, msgs)
	require.Error(t, err)
	assert.Equal(t, []int{0, 1}, failedIndexesFromBatchError(t, msgs, err))
	assert.Greater(t, writer.flushCallsFor(retryTableA), 1, "write should have been retried before cancellation")
}

func TestWriteBatchBoundedRetryNacksAfterMaxAttempts(t *testing.T) {
	writer := &flakyWriter{failing: true}
	output := newRetryTestOutput(t, writer, &testRejectSink{})
	output.groupRetryMaxAttempts = 2

	msgs := service.MessageBatch{
		newKafkaMessage(mustEventJSON(t, "e1", xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD), "t", 0, 1),
	}

	err := output.WriteBatch(context.Background(), msgs)
	require.Error(t, err)
	assert.Equal(t, []int{0}, failedIndexesFromBatchError(t, msgs, err))
	assert.Equal(t, 3, writer.flushCallsFor(retryTableA))
}

// limitedInput mimics the kafka_franz reader: single-message batches, blocking once its slots are unacked.
type limitedInput struct {
	slots chan struct{}
	seq   atomic.Int64
}

func (in *limitedInput) Connect(context.Context) error { return nil }

func (in *limitedInput) Close(context.Context) error { return nil }

func (in *limitedInput) ReadBatch(ctx context.Context) (service.MessageBatch, service.AckFunc, error) {
	select {
	case in.slots <- struct{}{}:
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	n := in.seq.Add(1) - 1

	payload, err := protojson.Marshal(&xatu.DecoratedEvent{
		Event: &xatu.Event{
			Id:       strconv.FormatInt(n, 10),
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD,
			DateTime: timestamppb.Now(),
		},
	})
	if err != nil {
		return nil, nil, err
	}

	return service.MessageBatch{newKafkaMessage(payload, "t", 0, n)}, func(_ context.Context, err error) error {
		if err == nil {
			<-in.slots
		}

		return nil
	}, nil
}

// TestStreamRecoversPromptlyAfterOutage drives a sink outage through a Benthos stream with nack auto-replay, as kafka_franz uses.
func TestStreamRecoversPromptlyAfterOutage(t *testing.T) {
	const backlog = 20

	writer := &flakyWriter{failing: true}
	output := newRetryTestOutput(t, writer, &testRejectSink{})
	output.groupRetryBaseDelay = 5 * time.Millisecond
	output.groupRetryMaxDelay = 20 * time.Millisecond

	input := &limitedInput{slots: make(chan struct{}, backlog)}

	env := service.NewEnvironment()
	require.NoError(t, env.RegisterBatchInput("limited_test", service.NewConfigSpec(),
		func(*service.ParsedConfig, *service.Resources) (service.BatchInput, error) {
			return service.AutoRetryNacksBatched(input), nil
		}))
	require.NoError(t, env.RegisterBatchOutput(benthosOutputType, service.NewConfigSpec(),
		func(*service.ParsedConfig, *service.Resources) (service.BatchOutput, service.BatchPolicy, int, error) {
			return output, service.BatchPolicy{Count: 100000, Period: "50ms"}, 64, nil
		}))

	builder := env.NewStreamBuilder()
	require.NoError(t, builder.SetYAML(fmt.Sprintf(`
input:
  limited_test: {}
output:
  %s: {}
logger:
  level: OFF
metrics:
  none: {}
http:
  enabled: false
shutdown_timeout: 1s
`, benthosOutputType)))

	stream, err := builder.Build()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})

	go func() {
		defer close(done)

		_ = stream.Run(ctx)
	}()

	t.Cleanup(func() {
		cancel()
		<-done
	})

	time.Sleep(600 * time.Millisecond)
	writer.setFailing(false)

	require.Eventually(t, func() bool {
		return len(writer.writtenTo(retryTableA)) > backlog
	}, 2*time.Second, 10*time.Millisecond, "backlog and fresh messages should flow promptly once writes succeed")

	written := writer.writtenTo(retryTableA)
	for i := range backlog {
		assert.Contains(t, written, strconv.Itoa(i))
	}
}
