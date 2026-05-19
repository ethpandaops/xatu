package http

import (
	"context"
	"fmt"
	"io"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
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
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type recordedHTTPRequest struct {
	traceparent string
	eventCount  int
}

type recordingHTTPServer struct {
	mu       sync.Mutex
	requests []recordedHTTPRequest
	server   *httptest.Server
}

func TestHTTPSinkProcessorPropagatesAndBatchesByTraceContext(t *testing.T) {
	setupHTTPTraceTest(t)

	recorder := startRecordingHTTPServer(t)
	sink := newTestHTTPSink(t, recorder.server.URL, 2)

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	ctx1, span1 := otel.Tracer("test").Start(context.Background(), "event-1")
	defer span1.End()

	ctx2, span2 := otel.Tracer("test").Start(context.Background(), "event-2")
	defer span2.End()

	require.NoError(t, sink.HandleNewDecoratedEvent(ctx1, newHTTPTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, "id-1")))
	require.NoError(t, sink.HandleNewDecoratedEvent(ctx2, newHTTPTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD, "id-2")))

	require.Eventually(t, func() bool {
		return len(recorder.snapshotRequests()) == 2
	}, time.Second, 10*time.Millisecond)

	requests := recorder.snapshotRequests()
	require.Len(t, requests, 2)
	assert.Equal(t, 1, requests[0].eventCount)
	assert.Equal(t, 1, requests[1].eventCount)

	sc1 := spanContextFromHTTPTraceparent(requests[0].traceparent)
	sc2 := spanContextFromHTTPTraceparent(requests[1].traceparent)

	require.True(t, sc1.IsValid())
	require.True(t, sc2.IsValid())
	assert.Equal(t, span1.SpanContext().TraceID(), sc1.TraceID())
	assert.Equal(t, span2.SpanContext().TraceID(), sc2.TraceID())
	assert.NotEqual(t, sc1.TraceID(), sc2.TraceID())
}

func TestHTTPSinkProcessorBatchesSharedTraceContext(t *testing.T) {
	setupHTTPTraceTest(t)

	recorder := startRecordingHTTPServer(t)
	sink := newTestHTTPSink(t, recorder.server.URL, 100)

	require.NoError(t, sink.Start(context.Background()))

	defer func() {
		require.NoError(t, sink.Stop(context.Background()))
	}()

	ctx, span := otel.Tracer("test").Start(context.Background(), "shared")
	defer span.End()

	events := make([]*xatu.DecoratedEvent, 0, 100)
	for i := range 100 {
		events = append(events, newHTTPTestEvent(xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK, fmt.Sprintf("id-%d", i)))
	}

	require.NoError(t, sink.HandleNewDecoratedEvents(ctx, events))

	require.Eventually(t, func() bool {
		requests := recorder.snapshotRequests()

		return len(requests) == 1 && requests[0].eventCount == 100
	}, time.Second, 10*time.Millisecond)

	requests := recorder.snapshotRequests()
	require.Len(t, requests, 1)
	assert.Equal(t, span.SpanContext().TraceID(), spanContextFromHTTPTraceparent(requests[0].traceparent).TraceID())
}

func startRecordingHTTPServer(t *testing.T) *recordingHTTPServer {
	t.Helper()

	recorder := &recordingHTTPServer{}

	recorder.server = httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(stdhttp.StatusInternalServerError)

			return
		}

		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, recordedHTTPRequest{
			traceparent: r.Header.Get("traceparent"),
			eventCount:  strings.Count(string(raw), "\n"),
		})
		recorder.mu.Unlock()

		w.WriteHeader(stdhttp.StatusOK)
	}))

	t.Cleanup(recorder.server.Close)

	return recorder
}

func (r *recordingHTTPServer) snapshotRequests() []recordedHTTPRequest {
	r.mu.Lock()
	defer r.mu.Unlock()

	return append([]recordedHTTPRequest(nil), r.requests...)
}

func newTestHTTPSink(t *testing.T, address string, batchSize int) *HTTP {
	t.Helper()

	sink, err := New("test", &Config{
		Address:            address,
		Headers:            map[string]string{},
		MaxQueueSize:       batchSize,
		BatchTimeout:       time.Hour,
		ExportTimeout:      5 * time.Second,
		MaxExportBatchSize: batchSize,
		Workers:            1,
	}, logrus.New(), &xatu.EventFilterConfig{}, processor.ShippingMethodAsync)
	require.NoError(t, err)

	return sink
}

func setupHTTPTraceTest(t *testing.T) {
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

func newHTTPTestEvent(name xatu.Event_Name, id string) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     name,
			Id:       id,
			DateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
		},
	}
}

func spanContextFromHTTPTraceparent(traceparent string) trace.SpanContext {
	return trace.SpanContextFromContext(
		otel.GetTextMapPropagator().Extract(
			context.Background(),
			propagation.MapCarrier{"traceparent": traceparent},
		),
	)
}
