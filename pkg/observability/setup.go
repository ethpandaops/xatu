package observability

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/sdk/trace"
)

// Setup bootstraps OTel traces and the W3C propagator for a component.
// Logs are emitted to stdout with trace_id/span_id fields and shipped
// out-of-process by a local agent. Returns a shutdown func.
func Setup(
	ctx context.Context,
	log *logrus.Logger,
	serviceName, serviceVersion string,
	traceCfg *TracingConfig,
) (shutdown func(context.Context) error, err error) {
	shutdownFuncs := make([]func(context.Context) error, 0, 1)

	shutdown = func(ctx context.Context) error {
		var sErr error

		for _, fn := range shutdownFuncs {
			sErr = errors.Join(sErr, fn(ctx))
		}

		shutdownFuncs = nil

		return sErr
	}

	if !traceCfg.Enabled {
		return shutdown, nil
	}

	log.WithContext(ctx).Info("OTel tracing enabled")

	res, err := NewResource(serviceName, serviceVersion)
	if err != nil {
		return shutdown, fmt.Errorf("create otel resource: %w", err)
	}

	RegisterPropagator()

	tp, err := NewHTTPTraceProvider(ctx, res, traceCfg.AsOTelOpts(),
		trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(traceCfg.Sampling.Rate))),
	)
	if err != nil {
		return shutdown, fmt.Errorf("create trace provider: %w", err)
	}

	traceShutdown, err := SetupOTelSDK(ctx, tp)
	if err != nil {
		return shutdown, fmt.Errorf("setup trace SDK: %w", err)
	}

	shutdownFuncs = append(shutdownFuncs, traceShutdown)

	return shutdown, nil
}
