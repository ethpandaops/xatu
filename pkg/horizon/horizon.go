package horizon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	oxatu "github.com/ethpandaops/xatu/pkg/output/xatu"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/google/uuid"
	perrors "github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/sdk/trace"
)

type Horizon struct {
	Config *Config

	sinks []output.Sink

	log logrus.FieldLogger

	id uuid.UUID

	metrics *Metrics

	shutdownFuncs []func(ctx context.Context) error

	overrides *Override
}

func New(ctx context.Context, log logrus.FieldLogger, config *Config, overrides *Override) (*Horizon, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	if overrides != nil {
		if err := config.ApplyOverrides(overrides, log); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	sinks, err := config.CreateSinks(log)
	if err != nil {
		return nil, err
	}

	return &Horizon{
		Config:        config,
		sinks:         sinks,
		log:           log,
		id:            uuid.New(),
		metrics:       NewMetrics("xatu_horizon"),
		shutdownFuncs: make([]func(ctx context.Context) error, 0),
		overrides:     overrides,
	}, nil
}

func (h *Horizon) Start(ctx context.Context) error {
	// Start tracing if enabled
	if h.Config.Tracing.Enabled {
		h.log.Info("Tracing enabled")

		res, err := observability.NewResource(xatu.WithModule(xatu.ModuleName_HORIZON), xatu.Short())
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing resource")
		}

		opts := []trace.TracerProviderOption{
			trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(h.Config.Tracing.Sampling.Rate))),
		}

		tracer, err := observability.NewHTTPTraceProvider(ctx,
			res,
			h.Config.Tracing.AsOTelOpts(),
			opts...,
		)
		if err != nil {
			return perrors.Wrap(err, "failed to create tracing provider")
		}

		shutdown, err := observability.SetupOTelSDK(ctx, tracer)
		if err != nil {
			return perrors.Wrap(err, "failed to setup tracing SDK")
		}

		h.shutdownFuncs = append(h.shutdownFuncs, shutdown)
	}

	if err := h.ServeMetrics(ctx); err != nil {
		return err
	}

	if h.Config.PProfAddr != nil {
		if err := h.ServePProf(ctx); err != nil {
			return err
		}
	}

	h.log.
		WithField("version", xatu.Full()).
		WithField("id", h.id.String()).
		Info("Starting Xatu in horizon mode 🌅")

	for _, sink := range h.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	if err := h.ApplyOverrideBeforeStartAfterCreation(ctx); err != nil {
		return fmt.Errorf("failed to apply overrides before start: %w", err)
	}

	cancel := make(chan os.Signal, 1)
	signal.Notify(cancel, syscall.SIGTERM, syscall.SIGINT)

	sig := <-cancel
	h.log.Printf("Caught signal: %v", sig)

	if err := h.Shutdown(ctx); err != nil {
		return err
	}

	return nil
}

func (h *Horizon) Shutdown(ctx context.Context) error {
	h.log.Printf("Shutting down")

	for _, sink := range h.sinks {
		if err := sink.Stop(ctx); err != nil {
			return err
		}
	}

	for _, fun := range h.shutdownFuncs {
		if err := fun(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (h *Horizon) ApplyOverrideBeforeStartAfterCreation(ctx context.Context) error {
	if h.overrides == nil {
		return nil
	}

	if h.overrides.XatuOutputAuth.Enabled {
		h.log.Info("Overriding output authorization on xatu sinks")

		for _, sink := range h.sinks {
			if sink.Type() == string(output.SinkTypeXatu) {
				xatuSink, ok := sink.(*oxatu.Xatu)
				if !ok {
					return perrors.New("failed to assert xatu sink")
				}

				h.log.WithField("sink_name", sink.Name()).Info("Overriding xatu output authorization")

				xatuSink.SetAuthorization(h.overrides.XatuOutputAuth.Value)
			}
		}
	}

	return nil
}

func (h *Horizon) ServeMetrics(ctx context.Context) error {
	go func() {
		sm := http.NewServeMux()
		sm.Handle("/metrics", promhttp.Handler())

		server := &http.Server{
			Addr:              h.Config.MetricsAddr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           sm,
		}

		h.log.Infof("Serving metrics at %s", h.Config.MetricsAddr)

		if err := server.ListenAndServe(); err != nil {
			h.log.Fatal(err)
		}
	}()

	return nil
}

func (h *Horizon) ServePProf(ctx context.Context) error {
	pprofServer := &http.Server{
		Addr:              *h.Config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	go func() {
		h.log.Infof("Serving pprof at %s", *h.Config.PProfAddr)

		if err := pprofServer.ListenAndServe(); err != nil {
			h.log.Fatal(err)
		}
	}()

	return nil
}
