package xatu

import (
	"context"
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
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/processor"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type recordedCreateEventsCall struct {
	traceparent string
	eventCount  int
}

type recordingIngester struct {
	pb.UnimplementedEventIngesterServer

	mu    sync.Mutex
	calls []recordedCreateEventsCall
}

func (r *recordingIngester) CreateEvents(ctx context.Context, req *pb.CreateEventsRequest) (*pb.CreateEventsResponse, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	traceparent := ""
	if values := md.Get("traceparent"); len(values) > 0 {
		traceparent = values[0]
	}

	r.mu.Lock()
	r.calls = append(r.calls, recordedCreateEventsCall{
		traceparent: traceparent,
		eventCount:  len(req.GetEvents()),
	})
	r.mu.Unlock()

	return &pb.CreateEventsResponse{
		EventsIngested: wrapperspb.UInt64(uint64(len(req.GetEvents()))),
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

func startRecordingIngester(t *testing.T) (string, *recordingIngester) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := grpc.NewServer()
	ingester := &recordingIngester{}
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
