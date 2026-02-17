package consumoor

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener/tables"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// Consumoor is the main service that consumes events from Kafka and
// writes them to ClickHouse.
type Consumoor struct {
	log    logrus.FieldLogger
	config *Config

	metrics *Metrics
	router  *Router
	writer  Writer
	stream  *service.Stream

	metricsServer *http.Server
	pprofServer   *http.Server
}

// New creates a new Consumoor service. Call Start() to run it.
func New(
	ctx context.Context,
	log logrus.FieldLogger,
	config *Config,
	overrides *Override,
) (*Consumoor, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	config.ApplyOverrides(overrides)

	metrics := NewMetrics("xatu")

	// Create the ClickHouse writer
	writer, err := NewWriter(
		log,
		&config.ClickHouse,
		metrics,
	)
	if err != nil {
		return nil, fmt.Errorf("creating clickhouse writer: %w", err)
	}

	// Create the router with all registered routes.
	registeredRoutes := buildRoutes()

	disabledEvents, err := config.DisabledEventEnums()
	if err != nil {
		return nil, fmt.Errorf("invalid disabledEvents config: %w", err)
	}

	router := NewRouter(log, registeredRoutes, disabledEvents, metrics)

	c := &Consumoor{
		log:     log.WithField("component", "consumoor"),
		config:  config,
		metrics: metrics,
		router:  router,
		writer:  writer,
	}

	stream, err := NewBenthosStream(
		log,
		config,
		metrics,
		router,
		writer,
	)
	if err != nil {
		return nil, fmt.Errorf("creating benthos stream: %w", err)
	}

	c.stream = stream

	return c, nil
}

// Start runs the consumoor service until the context is cancelled or
// a SIGINT/SIGTERM is received.
func (c *Consumoor) Start(ctx context.Context) error {
	nctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	c.log.Info("Consumoor started (benthos runtime)")

	g, gCtx := errgroup.WithContext(nctx)

	g.Go(func() error {
		return c.startMetrics(ctx)
	})

	if c.config.PProfAddr != nil {
		g.Go(func() error {
			return c.startPProf(ctx)
		})
	}

	g.Go(func() error {
		if err := c.stream.Run(gCtx); err != nil && !errors.Is(err, context.Canceled) {
			return fmt.Errorf("running benthos stream: %w", err)
		}

		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()

		return c.stop(ctx)
	})

	err := g.Wait()

	if err != nil && err != context.Canceled {
		return err
	}

	return nil
}

// stop gracefully shuts down all components.
func (c *Consumoor) stop(ctx context.Context) error {
	c.log.Info("Stopping consumoor")

	if c.stream != nil {
		if err := c.stream.StopWithin(30 * time.Second); err != nil &&
			err.Error() != "stream has not been run yet" {
			c.log.WithError(err).Error("Error stopping benthos stream")
		}
	}

	// Stop HTTP servers
	if c.metricsServer != nil {
		if err := c.metricsServer.Shutdown(ctx); err != nil {
			c.log.WithError(err).Error("Error stopping metrics server")
		}
	}

	if c.pprofServer != nil {
		if err := c.pprofServer.Shutdown(ctx); err != nil {
			c.log.WithError(err).Error("Error stopping pprof server")
		}
	}

	c.log.Info("Consumoor stopped")

	return nil
}

func (c *Consumoor) startMetrics(ctx context.Context) error {
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())

	c.log.WithField("addr", c.config.MetricsAddr).Info("Starting metrics server")

	c.metricsServer = &http.Server{
		Addr:              c.config.MetricsAddr,
		ReadHeaderTimeout: 15 * time.Second,
		Handler:           sm,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	if err := c.metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (c *Consumoor) startPProf(_ context.Context) error {
	c.log.WithField("addr", c.config.PProfAddr).Info("Starting pprof server")

	c.pprofServer = &http.Server{
		Addr:              *c.config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	if err := c.pprofServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}

	return nil
}

// buildRoutes returns all registered route implementations.
func buildRoutes() []flattener.Route {
	return tables.All()
}
