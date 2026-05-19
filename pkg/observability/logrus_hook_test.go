package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/trace"
)

func TestTraceContextHook_NoSpan(t *testing.T) {
	log := newJSONLogger(t)

	log.WithContext(context.Background()).Info("hello")

	buf, ok := log.Out.(*bytes.Buffer)
	require.True(t, ok)

	entry := decode(t, buf)
	_, hasTrace := entry["trace_id"]
	require.False(t, hasTrace, "trace_id should not be set when no span is active")
}

func TestTraceContextHook_WithSpan(t *testing.T) {
	tp := trace.NewTracerProvider()
	otel.SetTracerProvider(tp)

	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	log := newJSONLogger(t)

	ctx, span := tp.Tracer("test").Start(context.Background(), "op")
	defer span.End()

	log.WithContext(ctx).Info("hello")

	buf, ok := log.Out.(*bytes.Buffer)
	require.True(t, ok)

	entry := decode(t, buf)

	require.Equal(t, span.SpanContext().TraceID().String(), entry["trace_id"])
	require.Equal(t, span.SpanContext().SpanID().String(), entry["span_id"])
}

func newJSONLogger(t *testing.T) *logrus.Logger {
	t.Helper()

	l := logrus.New()
	l.SetFormatter(&logrus.JSONFormatter{})
	l.SetOutput(&bytes.Buffer{})

	InstallTraceContextHook(l)

	return l
}

func decode(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()

	out := make(map[string]any, 8)

	require.NoError(t, json.Unmarshal(buf.Bytes(), &out))

	return out
}
