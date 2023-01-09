package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/ethpandaops/xatu/pkg/server/service"
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
}

func NewXatu(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Xatu, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	services, err := service.CreateGRPCServices(ctx, log, &conf.Services)
	if err != nil {
		return nil, err
	}

	return &Xatu{
		config:   conf,
		log:      log.WithField("component", "server"),
		ctx:      ctx,
		services: services,
	}, nil
}

func (x *Xatu) Start(ctx context.Context) error {
	nctx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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
