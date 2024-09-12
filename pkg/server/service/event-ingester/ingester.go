package eventingester

import (
	"context"
	"errors"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	ServiceType = "event-ingester"
)

type Ingester struct {
	xatu.UnimplementedEventIngesterServer

	log     logrus.FieldLogger
	config  *Config
	handler *Handler

	sinks []output.Sink
}

func NewIngester(ctx context.Context, log logrus.FieldLogger, conf *Config, clockDrift *time.Duration, geoipProvider geoip.Provider, cache store.Cache) (*Ingester, error) {
	e := &Ingester{
		log:     log.WithField("server/module", ServiceType),
		config:  conf,
		handler: NewHandler(log, clockDrift, geoipProvider, cache),
	}

	sinks, err := e.CreateSinks()
	if err != nil {
		return e, err
	}

	e.sinks = sinks

	return e, nil
}

func (e *Ingester) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("Starting module")

	xatu.RegisterEventIngesterServer(grpcServer, e)

	for _, sink := range e.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (e *Ingester) Stop(ctx context.Context) error {
	e.log.Info("Stopping module")

	for _, sink := range e.sinks {
		if err := sink.Stop(ctx); err != nil {
			return status.Error(codes.Internal, err.Error())
		}
	}

	return nil
}

func (e *Ingester) CreateEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	e.log.WithField("events", len(req.Events)).Debug("Received batch of events")

	// TODO(sam.calder-mason): Derive client id/name from the request jwt
	clientID := "unknown"

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("failed to get metadata from context")
	}

	if e.config.AuthorizationSecret != "" {
		authorization := md.Get("authorization")
		if len(authorization) > 0 {
			if authorization[0] != e.config.AuthorizationSecret {
				return nil, status.Error(codes.Unauthenticated, "invalid authorization secret")
			}
		}
	}

	filteredEvents, err := e.handler.Events(ctx, clientID, req.Events)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	for _, sink := range e.sinks {
		if err := sink.HandleNewDecoratedEvents(ctx, filteredEvents); err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &xatu.CreateEventsResponse{}, nil
}

func (e *Ingester) CreateSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(e.config.Outputs))

	for i, out := range e.config.Outputs {
		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			e.log,
			out.FilterConfig,
			processor.ShippingMethodSync,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}
