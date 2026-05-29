package xatu

import (
	"context"
	"fmt"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/processor"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type recordedCreateEventsCall struct {
	traceparent string
	eventCount  int
	eventIDs    []string
	eventNames  []pb.Event_Name
}

type recordingIngester struct {
	pb.UnimplementedEventIngesterServer

	mu     sync.Mutex
	calls  []recordedCreateEventsCall
	errFor func(recordedCreateEventsCall) error
}

func (r *recordingIngester) CreateEvents(ctx context.Context, req *pb.CreateEventsRequest) (*pb.CreateEventsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	traceparent := ""
	if values := md.Get("traceparent"); len(values) > 0 {
		traceparent = values[0]
	}

	events := req.GetEvents()
	eventIDs := make([]string, 0, len(events))
	eventNames := make([]pb.Event_Name, 0, len(events))

	for _, event := range events {
		eventIDs = append(eventIDs, event.GetEvent().GetId())
		eventNames = append(eventNames, event.GetEvent().GetName())
	}

	call := recordedCreateEventsCall{
		traceparent: traceparent,
		eventCount:  len(events),
		eventIDs:    eventIDs,
		eventNames:  eventNames,
	}

	r.mu.Lock()
	r.calls = append(r.calls, call)
	r.mu.Unlock()

	if r.errFor != nil {
		if err := r.errFor(call); err != nil {
			return nil, err
		}
	}

	return &pb.CreateEventsResponse{
		EventsIngested: wrapperspb.UInt64(uint64(len(events))),
	}, nil
}

func (r *recordingIngester) snapshotCalls() []recordedCreateEventsCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]recordedCreateEventsCall(nil), r.calls...)
}

func TestXatuSinkProcessorPropagatesPerItemContexts(t *testing.T) {
	setupXatuTraceTest(t)

	addr, ingester := startRecordingIngester(t)

	sink, err := New("test", &Config{
		Address:            addr,
		Headers:            map[string]string{},
		MaxQueueSize:       10,
		BatchTimeout:       time.Hour,
		ExportTimeout:      5 * time.Second,
		MaxExportBatchSize: 2,
		Workers:            1,
	}, logrus.New(), &pb.EventFilterConfig{}, processor.ShippingMethodAsync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	ctx1, span1 := otel.Tracer("test").Start(context.Background(), "event-1")
	defer span1.End()

	ctx2, span2 := otel.Tracer("test").Start(context.Background(), "event-2")
	defer span2.End()

	require.NoError(t, sink.HandleNewDecoratedEvent(ctx1, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1")))
	require.NoError(t, sink.HandleNewDecoratedEvent(ctx2, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2")))

	require.Eventually(t, func() bool {
		return len(ingester.snapshotCalls()) == 2
	}, time.Second, 10*time.Millisecond)

	calls := ingester.snapshotCalls()
	require.Len(t, calls, 2)
	assert.Equal(t, 1, calls[0].eventCount)
	assert.Equal(t, 1, calls[1].eventCount)

	sc1 := spanContextFromTraceparent(calls[0].traceparent)
	sc2 := spanContextFromTraceparent(calls[1].traceparent)

	require.True(t, sc1.IsValid())
	require.True(t, sc2.IsValid())
	assert.Equal(t, span1.SpanContext().TraceID(), sc1.TraceID())
	assert.Equal(t, span2.SpanContext().TraceID(), sc2.TraceID())
	assert.NotEqual(t, sc1.TraceID(), sc2.TraceID())
}

func TestXatuSinkProcessorBatchesSharedContext(t *testing.T) {
	setupXatuTraceTest(t)

	addr, ingester := startRecordingIngester(t)

	sink, err := New("test", &Config{
		Address:            addr,
		Headers:            map[string]string{},
		MaxQueueSize:       100,
		BatchTimeout:       time.Hour,
		ExportTimeout:      5 * time.Second,
		MaxExportBatchSize: 100,
		Workers:            1,
	}, logrus.New(), &pb.EventFilterConfig{}, processor.ShippingMethodAsync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	ctx, span := otel.Tracer("test").Start(context.Background(), "shared")
	defer span.End()

	events := make([]*pb.DecoratedEvent, 0, 100)
	for i := range 100 {
		events = append(events, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, fmt.Sprintf("id-%d", i)))
	}

	require.NoError(t, sink.HandleNewDecoratedEvents(ctx, events))

	require.Eventually(t, func() bool {
		calls := ingester.snapshotCalls()

		return len(calls) == 1 && calls[0].eventCount == 100
	}, time.Second, 10*time.Millisecond)

	calls := ingester.snapshotCalls()
	require.Len(t, calls, 1)
	assert.Equal(t, span.SpanContext().TraceID(), spanContextFromTraceparent(calls[0].traceparent).TraceID())
}

func TestXatuSinkProcessorPreservesSparseMixedTraceGroups(t *testing.T) {
	setupXatuTraceTest(t)

	addr, ingester := startRecordingIngester(t)

	sink, err := New("test", &Config{
		Address:            addr,
		Headers:            map[string]string{},
		MaxQueueSize:       12,
		BatchTimeout:       time.Hour,
		ExportTimeout:      5 * time.Second,
		MaxExportBatchSize: 12,
		Workers:            1,
	}, logrus.New(), &pb.EventFilterConfig{}, processor.ShippingMethodAsync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	for i := range 10 {
		require.NoError(t, sink.HandleNewDecoratedEvent(
			context.Background(),
			newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2, fmt.Sprintf("attestation-%d", i)),
		))
	}

	headCtx, headSpan := otel.Tracer("test").Start(context.Background(), "head")
	defer headSpan.End()

	blockCtx, blockSpan := otel.Tracer("test").Start(context.Background(), "block")
	defer blockSpan.End()

	require.NoError(t, sink.HandleNewDecoratedEvent(headCtx, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2, "head-1")))
	require.NoError(t, sink.HandleNewDecoratedEvent(blockCtx, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2, "block-1")))

	require.Eventually(t, func() bool {
		return len(ingester.snapshotCalls()) == 3
	}, time.Second, 10*time.Millisecond)

	calls := ingester.snapshotCalls()
	require.Len(t, calls, 3)

	assert.Equal(t, 10, calls[0].eventCount)
	assert.Equal(t, 1, calls[1].eventCount)
	assert.Equal(t, 1, calls[2].eventCount)
	assert.Equal(t, []string{"head-1"}, calls[1].eventIDs)
	assert.Equal(t, []string{"block-1"}, calls[2].eventIDs)
	assert.Equal(t, []pb.Event_Name{pb.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2}, calls[1].eventNames)
	assert.Equal(t, []pb.Event_Name{pb.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2}, calls[2].eventNames)

	assert.Equal(t, headSpan.SpanContext().TraceID(), spanContextFromTraceparent(calls[1].traceparent).TraceID())
	assert.Equal(t, blockSpan.SpanContext().TraceID(), spanContextFromTraceparent(calls[2].traceparent).TraceID())
}

func TestXatuSinkProcessorAttemptsSparseGroupsAfterEarlierGroupFailure(t *testing.T) {
	setupXatuTraceTest(t)

	addr, ingester := startRecordingIngesterWithError(t, func(call recordedCreateEventsCall) error {
		if call.eventCount == 10 {
			return status.Error(codes.Internal, "first group failed")
		}

		return nil
	})

	sink, err := New("test", &Config{
		Address:            addr,
		Headers:            map[string]string{},
		MaxQueueSize:       12,
		BatchTimeout:       time.Hour,
		ExportTimeout:      5 * time.Second,
		MaxExportBatchSize: 12,
		Workers:            1,
	}, logrus.New(), &pb.EventFilterConfig{}, processor.ShippingMethodAsync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	for i := range 10 {
		require.NoError(t, sink.HandleNewDecoratedEvent(
			context.Background(),
			newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION_V2, fmt.Sprintf("attestation-%d", i)),
		))
	}

	headCtx, headSpan := otel.Tracer("test").Start(context.Background(), "head")
	defer headSpan.End()

	blockCtx, blockSpan := otel.Tracer("test").Start(context.Background(), "block")
	defer blockSpan.End()

	require.NoError(t, sink.HandleNewDecoratedEvent(headCtx, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2, "head-1")))
	require.NoError(t, sink.HandleNewDecoratedEvent(blockCtx, newTestEvent(pb.Event_BEACON_API_ETH_V1_EVENTS_BLOCK_V2, "block-1")))

	require.Eventually(t, func() bool {
		return len(ingester.snapshotCalls()) == 3
	}, time.Second, 10*time.Millisecond)

	calls := ingester.snapshotCalls()
	require.Len(t, calls, 3)

	assert.Equal(t, []string{"head-1"}, calls[1].eventIDs)
	assert.Equal(t, []string{"block-1"}, calls[2].eventIDs)
	assert.Equal(t, headSpan.SpanContext().TraceID(), spanContextFromTraceparent(calls[1].traceparent).TraceID())
	assert.Equal(t, blockSpan.SpanContext().TraceID(), spanContextFromTraceparent(calls[2].traceparent).TraceID())
}

func startRecordingIngester(t *testing.T) (string, *recordingIngester) {
	t.Helper()

	return startRecordingIngesterWithError(t, nil)
}

func startRecordingIngesterWithError(t *testing.T, errFor func(recordedCreateEventsCall) error) (string, *recordingIngester) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	ingester := &recordingIngester{errFor: errFor}
	pb.RegisterEventIngesterServer(server, ingester)

	go func() {
		_ = server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
	})

	return listener.Addr().String(), ingester
}

func setupXatuTraceTest(t *testing.T) {
	t.Helper()

	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(tracetest.NewSpanRecorder()),
	)

	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	t.Cleanup(func() {
		require.NoError(t, provider.Shutdown(context.Background()))
		otel.SetTracerProvider(previousProvider)
		otel.SetTextMapPropagator(previousPropagator)
	})
}

func newTestEvent(name pb.Event_Name, id string) *pb.DecoratedEvent {
	return &pb.DecoratedEvent{
		Event: &pb.Event{
			Name:     name,
			Id:       id,
			DateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
		},
	}
}

func spanContextFromTraceparent(traceparent string) trace.SpanContext {
	return trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(
			context.Background(),
			propagation.MapCarrier{"traceparent": traceparent},
		),
	)
}
