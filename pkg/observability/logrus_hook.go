package observability

import (
	"context"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/trace"
)

// ContextualLogger extends logrus.FieldLogger with WithContext so call
// sites that have a context.Context can attach it to log entries. Both
// *logrus.Logger and *logrus.Entry satisfy this interface.
type ContextualLogger interface {
	logrus.FieldLogger
	WithContext(ctx context.Context) *logrus.Entry
}

// TraceContextHook is a logrus hook that stamps the active OTel span's
// trace_id and span_id onto each log entry that carries a context.
type TraceContextHook struct{}

// Levels reports the levels this hook fires on.
func (TraceContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// Fire enriches the entry with trace_id and span_id when its context
// carries a valid span.
func (TraceContextHook) Fire(entry *logrus.Entry) error {
	if entry.Context == nil {
		return nil
	}

	sc := trace.SpanContextFromContext(entry.Context)
	if !sc.IsValid() {
		return nil
	}

	if entry.Data == nil {
		entry.Data = make(logrus.Fields, 2)
	}

	entry.Data["trace_id"] = sc.TraceID().String()
	entry.Data["span_id"] = sc.SpanID().String()

	return nil
}

// InstallTraceContextHook attaches the trace-context hook to a logger.
// Call once at startup; Logrus does not deduplicate hooks.
func InstallTraceContextHook(log *logrus.Logger) {
	log.AddHook(TraceContextHook{})
}
