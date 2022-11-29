package server

import (
	"context"
	"fmt"
	"net"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	//nolint:blank-imports // Required for grpc.WithCompression
	_ "google.golang.org/grpc/encoding/gzip"
)

type Xatu struct {
	xatu.UnimplementedXatuServer

	ctx    context.Context
	log    logrus.FieldLogger
	config *Config

	sinks []output.Sink
}

func NewXatu(ctx context.Context, log logrus.FieldLogger, conf *Config) *Xatu {
	return &Xatu{
		config: conf,
		log:    log.WithField("component", "server"),
		ctx:    ctx,
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

	lis, err := net.Listen("tcp", x.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %v", err)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	xatu.RegisterXatuServer(grpcServer, x)

	x.log.WithField("addr", x.config.Addr).Info("Starting gRPC server")

	return grpcServer.Serve(lis)
}

func (x *Xatu) CreateEvents(ctx context.Context, in *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	x.log.WithField("events", len(in.Events)).Debug("Received batch of events")

	for _, sink := range x.sinks {
		for _, event := range in.Events {
			if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
				return nil, err
			}
		}
	}

	return &xatu.CreateEventsResponse{}, nil
}
