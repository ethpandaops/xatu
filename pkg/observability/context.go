package observability

import (
	"context"

	"go.opentelemetry.io/otel/trace"
)

func MergeContexts(parent context.Context, contexts ...context.Context) context.Context {
	spanContexts := make([]trace.SpanContext, 0, len(contexts))

	for _, ctx := range contexts {
		if spanContext := trace.SpanContextFromContext(ctx); spanContext.IsValid() {
			spanContexts = append(spanContexts, spanContext)
		}
	}

	if len(spanContexts) == 0 {
		return parent
	}

	// Create a new span that links to all the original spans
	tracer := Tracer()

	links := make([]trace.Link, len(spanContexts))
	for i, spanContext := range spanContexts {
		links[i] = trace.Link{SpanContext: spanContext}
	}

	ctx, span := tracer.Start(parent, "Observability.MergedSpan", trace.WithLinks(links...))
	defer span.End()

	return ctx
}
