package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
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
}

func NewXatu(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Xatu, error) {
	if err := conf.Validate(); err != nil {
		return nil, err
	}

	services, err := service.CreateGRPCServices(ctx, log, conf.Services)
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
	g, _ := errgroup.WithContext(ctx)

	g.Go(x.startMetrics)
	g.Go(x.startGrpcServer)

	return g.Wait()
}

func (x *Xatu) startGrpcServer() error {
	lis, err := net.Listen("tcp", x.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	opts := []grpc.ServerOption{
		grpc.StreamInterceptor(grpc_prometheus.StreamServerInterceptor),
		grpc.UnaryInterceptor(grpc_prometheus.UnaryServerInterceptor),
	}
	grpcServer := grpc.NewServer(opts...)

	for _, s := range x.services {
		if err := s.Start(grpcServer); err != nil {
			return err
		}
	}

	grpc_prometheus.Register(grpcServer)

	x.log.WithField("addr", x.config.Addr).Info("Starting gRPC server")

	return grpcServer.Serve(lis)
}

func (x *Xatu) startMetrics() error {
	http.Handle("/metrics", promhttp.Handler())

	x.log.WithField("addr", x.config.MetricsAddr).Info("Starting metrics server")

	server := &http.Server{
		Addr:              x.config.MetricsAddr,
		ReadHeaderTimeout: 15 * time.Second,
	}

	return server.ListenAndServe()
}
