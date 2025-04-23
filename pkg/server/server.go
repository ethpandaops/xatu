package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	//nolint:gosec // only exposed if pprofAddr config is set
	_ "net/http/pprof"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/service"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/go-co-op/gocron/v2"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/sdk/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	_ "google.golang.org/grpc/encoding/gzip"
)

type Xatu struct {
	log    logrus.FieldLogger
	config *Config

	services []service.GRPCService

	grpcServer    *grpc.Server
	metricsServer *http.Server
	pprofServer   *http.Server
	healthServer  *health.Server

	persistence   *persistence.Client
	cache         store.Cache
	geoipProvider geoip.Provider

	clockDrift *time.Duration

	overrides *Override

	shutdownFuncs []func(ctx context.Context) error
}

func NewXatu(ctx context.Context, log logrus.FieldLogger, conf *Config, o *Override) (*Xatu, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	if o != nil {
		if err := conf.ApplyOverrides(o, log); err != nil {
			return nil, fmt.Errorf("failed to apply overrides: %w", err)
		}
	}

	clockDrift := time.Duration(0)

	var p *persistence.Client

	if conf.Persistence.Enabled {
		var err error

		p, err = persistence.NewClient(ctx, log, &conf.Persistence)
		if err != nil {
			return nil, err
		}
	}

	c, err := store.NewCache(conf.Store.Type, conf.Store.Config, log)
	if err != nil {
		return nil, err
	}

	var g geoip.Provider
	if conf.GeoIP.Enabled {
		g, err = geoip.NewProvider(conf.GeoIP.Type, conf.GeoIP.Config, log)
		if err != nil {
			return nil, err
		}
	}

	healthServer := health.NewServer()

	xatuServer := &Xatu{
		config:        conf,
		log:           log.WithField("component", "server"),
		persistence:   p,
		cache:         c,
		geoipProvider: g,
		clockDrift:    &clockDrift,
		shutdownFuncs: []func(ctx context.Context) error{},
		overrides:     o,
		healthServer:  healthServer,
	}

	// Create services
	services, err := service.CreateGRPCServices(ctx, log, &conf.Services, &clockDrift, p, c, g, healthServer)
	if err != nil {
		return nil, err
	}

	xatuServer.services = services

	return xatuServer, nil
}

func (x *Xatu) Start(ctx context.Context) error {
	nctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Start tracing if enabled
	if x.config.Tracing.Enabled {
		x.log.Info("Tracing enabled")

		res, err := observability.NewResource(xatu.WithModule(xatu.ModuleName_SERVER), xatu.Short())
		if err != nil {
			return errors.Wrap(err, "failed to create tracing resource")
		}

		opts := []trace.TracerProviderOption{
			trace.WithSampler(trace.ParentBased(trace.TraceIDRatioBased(x.config.Tracing.Sampling.Rate))),
		}

		tracer, err := observability.NewHTTPTraceProvider(ctx,
			res,
			x.config.Tracing.AsOTelOpts(),
			opts...,
		)
		if err != nil {
			return errors.Wrap(err, "failed to create tracing provider")
		}

		shutdown, err := observability.SetupOTelSDK(ctx, tracer)
		if err != nil {
			return errors.Wrap(err, "failed to setup tracing SDK")
		}

		x.shutdownFuncs = append(x.shutdownFuncs, shutdown)
	}

	if err := x.startCrons(ctx); err != nil {
		x.log.WithError(err).Fatal("Failed to start crons")
	}

	if x.config.Persistence.Enabled {
		if err := x.persistence.Start(ctx); err != nil {
			return err
		}
	}

	if err := x.cache.Start(ctx); err != nil {
		return err
	}

	if x.config.GeoIP.Enabled {
		if err := x.geoipProvider.Start(ctx); err != nil {
			return err
		}
	}

	g, gCtx := errgroup.WithContext(nctx)

	g.Go(func() error {
		if err := x.startMetrics(ctx); err != nil {
			if err != http.ErrServerClosed {
				return err
			}
		}

		return nil
	})

	if x.config.PProfAddr != nil {
		g.Go(func() error {
			if err := x.startPProf(ctx); err != nil {
				if err != http.ErrServerClosed {
					return err
				}
			}

			return nil
		})
	}

	g.Go(func() error {
		if err := x.startGrpcServer(ctx); err != nil {
			return err
		}

		return nil
	})
	g.Go(func() error {
		<-gCtx.Done()

		if err := x.stop(ctx); err != nil {
			return err
		}

		return nil
	})

	err := g.Wait()

	if err != context.Canceled {
		return err
	}

	return nil
}

func (x *Xatu) stop(ctx context.Context) error {
	x.log.WithField("pre_stop_sleep_seconds", x.config.PreStopSleepSeconds).Info("Stopping server")

	time.Sleep(time.Duration(x.config.PreStopSleepSeconds) * time.Second)

	if x.healthServer != nil {
		// Mark all services as NOT_SERVING during shutdown
		x.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
	}

	if x.grpcServer != nil {
		x.grpcServer.GracefulStop()
	}

	for _, s := range x.services {
		if err := s.Stop(ctx); err != nil {
			return err
		}
	}

	if x.config.Persistence.Enabled && x.persistence != nil {
		if err := x.persistence.Stop(ctx); err != nil {
			return err
		}
	}

	if x.cache != nil {
		if err := x.cache.Stop(ctx); err != nil {
			return err
		}
	}

	if x.config.GeoIP.Enabled {
		if err := x.geoipProvider.Stop(ctx); err != nil {
			return err
		}
	}

	for _, f := range x.shutdownFuncs {
		if err := f(ctx); err != nil {
			return err
		}
	}

	if x.pprofServer != nil {
		if err := x.pprofServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	if x.metricsServer != nil {
		if err := x.metricsServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	x.log.Info("Server stopped")

	return nil
}

func (x *Xatu) startGrpcServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", x.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	mb100 := 1024 * 1024 * 100

	grpc_prometheus.EnableHandlingTimeHistogram(
		grpc_prometheus.WithHistogramBuckets(
			[]float64{
				0.01, 0.03, 0.1, 0.3, 1, 3, 6, 9, 12, 15, 18, 21, 24, 27, 30, 33,
			},
		),
	)

	// MaxConnectionAge/MaxConnectionAgeGrace should exceed NGINX's grpc_read_timeout and grpc_send_timeout to
	// prevent NGINX from terminating connections before the server's age limit is reached.
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(mb100),
		grpc.ChainStreamInterceptor(
			grpc.StreamServerInterceptor(grpc_prometheus.StreamServerInterceptor),
		),
		grpc.ChainUnaryInterceptor(
			grpc.UnaryServerInterceptor(grpc_prometheus.UnaryServerInterceptor),
			func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
				start := time.Now()

				resp, err := handler(ctx, req)
				if err != nil {
					x.log.
						WithField("method", info.FullMethod).
						WithField("duration", time.Since(start)).
						WithError(err).
						Error("RPC Error")
				}

				return resp, err
			},
		),
	}

	if x.config.KeepaliveParams != nil && x.config.KeepaliveParams.Enabled != nil && *x.config.KeepaliveParams.Enabled {
		keepaliveParams, err := x.config.KeepaliveParams.ToGRPCKeepaliveParams()
		if err != nil {
			return fmt.Errorf("failed to convert keepalive params: %w", err)
		}

		x.log.WithField("params", x.config.KeepaliveParams.String()).Info("Enabling keepalive")

		opts = append(opts, grpc.KeepaliveParams(keepaliveParams))
	} else {
		x.log.Info("Keepalive disabled")
	}

	x.grpcServer = grpc.NewServer(opts...)

	// Register the health check service
	healthpb.RegisterHealthServer(x.grpcServer, x.healthServer)

	// Set the overall health to SERVING
	x.healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)

	for _, s := range x.services {
		if err := s.Start(ctx, x.grpcServer); err != nil {
			return err
		}
	}

	// Register reflection service on gRPC server if enabled.
	if x.config.GRPCReflection {
		reflection.Register(x.grpcServer)
	}

	grpc_prometheus.Register(x.grpcServer)

	x.log.WithField("addr", x.config.Addr).Info("Starting gRPC server with health checks enabled")

	return x.grpcServer.Serve(lis)
}

func (x *Xatu) startMetrics(ctx context.Context) error {
	sm := http.NewServeMux()
	sm.Handle("/metrics", promhttp.Handler())

	x.log.WithField("addr", x.config.MetricsAddr).Info("Starting metrics server")

	x.metricsServer = &http.Server{
		Addr:              x.config.MetricsAddr,
		ReadHeaderTimeout: 15 * time.Second,
		Handler:           sm,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	return x.metricsServer.ListenAndServe()
}

func (x *Xatu) startPProf(_ context.Context) error {
	x.log.WithField("addr", x.config.PProfAddr).Info("Starting pprof server")

	x.pprofServer = &http.Server{
		Addr:              *x.config.PProfAddr,
		ReadHeaderTimeout: 120 * time.Second,
	}

	return x.pprofServer.ListenAndServe()
}

func (x *Xatu) startCrons(ctx context.Context) error {
	c, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	if _, err := c.NewJob(
		gocron.DurationJob(5*time.Minute),
		gocron.NewTask(
			func(ctx context.Context) {
				if err := x.syncClockDrift(ctx); err != nil {
					x.log.WithError(err).Error("Failed to sync clock drift")
				}
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	c.Start()

	return nil
}

func (x *Xatu) syncClockDrift(_ context.Context) error {
	response, err := ntp.Query(x.config.NTPServer)
	if err != nil {
		return err
	}

	err = response.Validate()
	if err != nil {
		return err
	}

	*x.clockDrift = response.ClockOffset
	x.log.WithField("drift", *x.clockDrift).Info("Updated clock drift")

	return err
}
