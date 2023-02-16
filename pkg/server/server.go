package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/beevik/ntp"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/persistence"
	"github.com/ethpandaops/xatu/pkg/server/service"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/go-co-op/gocron"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	//nolint:blank-imports // Required for grpc.WithCompression
	_ "google.golang.org/grpc/encoding/gzip"
)

type Xatu struct {
	ctx     context.Context
	log     logrus.FieldLogger
	config  *Config
	metrics *Metrics

	services []service.GRPCService

	grpcServer    *grpc.Server
	metricsServer *http.Server

	persistence   *persistence.Client
	cache         store.Cache
	geoipProvider geoip.Provider

	clockDrift *time.Duration
}

func NewXatu(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Xatu, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
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

	services, err := service.CreateGRPCServices(ctx, log, &conf.Services, &clockDrift, p, c, g)
	if err != nil {
		return nil, err
	}

	return &Xatu{
		config:        conf,
		log:           log.WithField("component", "server"),
		ctx:           ctx,
		persistence:   p,
		cache:         c,
		geoipProvider: g,
		services:      services,
		clockDrift:    &clockDrift,
	}, nil
}

func (x *Xatu) Start(ctx context.Context) error {
	nctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
	for _, s := range x.services {
		if err := s.Stop(ctx); err != nil {
			return err
		}
	}

	if x.grpcServer != nil {
		x.grpcServer.GracefulStop()
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

	if x.metricsServer != nil {
		if err := x.metricsServer.Shutdown(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (x *Xatu) startGrpcServer(ctx context.Context) error {
	lis, err := net.Listen("tcp", x.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}
	x.grpcServer = grpc.NewServer(opts...)

	for _, s := range x.services {
		if err := s.Start(ctx, x.grpcServer); err != nil {
			return err
		}
	}

	grpc_prometheus.Register(x.grpcServer)

	x.log.WithField("addr", x.config.Addr).Info("Starting gRPC server")

	return x.grpcServer.Serve(lis)
}

func (x *Xatu) startMetrics(ctx context.Context) error {
	http.Handle("/metrics", promhttp.Handler())

	x.log.WithField("addr", x.config.MetricsAddr).Info("Starting metrics server")

	x.metricsServer = &http.Server{
		Addr:              x.config.MetricsAddr,
		ReadHeaderTimeout: 15 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	return x.metricsServer.ListenAndServe()
}

func (x *Xatu) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5m").Do(func() {
		if err := x.syncClockDrift(ctx); err != nil {
			x.log.WithError(err).Error("Failed to sync clock drift")
		}
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}

func (x *Xatu) syncClockDrift(ctx context.Context) error {
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
