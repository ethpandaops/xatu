package observability

import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

// RegisterPropagator installs the W3C trace-context + baggage propagator
// as the global OTel TextMapPropagator.
func RegisterPropagator() {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}
