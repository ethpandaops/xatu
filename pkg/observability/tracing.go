package observability

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
	otrace "go.opentelemetry.io/otel/trace"
)

func Tracer() otrace.Tracer {
	return otel.GetTracerProvider().Tracer(fmt.Sprintf("%s/observability", xatu.Full()))
}

// SetupOTelSDK bootstraps the OpenTelemetry pipeline.
// If it does not return an error, make sure to call shutdown for proper cleanup.
func SetupOTelSDK(ctx context.Context, tracerProvider *trace.TracerProvider) (shutdown func(context.Context) error, err error) {
	var shutdownFuncs []func(context.Context) error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown = func(ctx context.Context) error {
		var err error

		for _, fn := range shutdownFuncs {
			err = errors.Join(err, fn(ctx))
		}

		shutdownFuncs = nil

		return err
	}

	shutdownFuncs = append(shutdownFuncs, tracerProvider.Shutdown)
	otel.SetTracerProvider(tracerProvider)

	return shutdown, nil
}

func NewResource(serviceName, serviceVersion string) (*resource.Resource, error) {
	return resource.Merge(resource.Default(),
		resource.NewWithAttributes(semconv.SchemaURL,
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
}

func NewHTTPTraceProvider(ctx context.Context, res *resource.Resource, httpOpts []otlptracehttp.Option, opts ...trace.TracerProviderOption) (*trace.TracerProvider, error) {
	client := otlptracehttp.NewClient(httpOpts...)

	exporter, err := otlptrace.New(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}

	options := append([]trace.TracerProviderOption{
		trace.WithBatcher(exporter),
		trace.WithResource(res),
	}, opts...)

	traceProvider := trace.NewTracerProvider(options...)

	return traceProvider, nil
}
