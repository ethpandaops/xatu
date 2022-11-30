package server

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"

	//nolint:blank-imports // Required for grpc.WithCompression
	_ "google.golang.org/grpc/encoding/gzip"
)

type Xatu struct {
	xatu.UnimplementedXatuServer

	ctx     context.Context
	log     logrus.FieldLogger
	config  *Config
	metrics *Metrics

	sinks []output.Sink
}

func NewXatu(ctx context.Context, log logrus.FieldLogger, conf *Config) *Xatu {
	return &Xatu{
		config:  conf,
		log:     log.WithField("component", "server"),
		ctx:     ctx,
		metrics: NewMetrics("xatu_server"),
	}
}

func (x *Xatu) CreateSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(x.config.Outputs))

	for i, out := range x.config.Outputs {
		sink, err := output.NewSink(out.SinkType, out.Config, x.log)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

func (x *Xatu) Start(ctx context.Context) error {
	sinks, err := x.CreateSinks()
	if err != nil {
		return err
	}

	x.sinks = sinks

	g, ctx := errgroup.WithContext(ctx)

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
	xatu.RegisterXatuServer(grpcServer, x)

	grpc_prometheus.Register(grpcServer)

	x.log.WithField("addr", x.config.Addr).Info("Starting gRPC server")

	return grpcServer.Serve(lis)
}

func (x *Xatu) startMetrics() error {
	http.Handle("/metrics", promhttp.Handler())

	x.log.WithField("addr", x.config.MetricsAddr).Info("Starting metrics server")

	return http.ListenAndServe(x.config.MetricsAddr, nil)
}

func (x *Xatu) CreateEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	x.log.WithField("events", len(req.Events)).Debug("Received batch of events")

	// TODO(sam.calder-mason): Add clock drift
	receivedAt := timestamppb.New(time.Now())

	p, _ := peer.FromContext(ctx)

	for _, event := range req.Events {
		// TODO(sam.calder-mason): Validate event
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		x.metrics.AddDecoratedEventReceived(1, event.Meta.Client.Event.Name.String(), "unknown")

		event.Meta.Server = &xatu.ServerMeta{
			Event: &xatu.ServerMeta_Event{
				DateTime: receivedAt,
			},
			Client: &xatu.ServerMeta_Client{
				IP: p.Addr.String(),
			},
		}
	}

	for _, sink := range x.sinks {
		for _, event := range req.Events {
			if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
				return nil, err
			}
		}
	}

	return &xatu.CreateEventsResponse{}, nil
}
