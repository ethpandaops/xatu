package eventingester

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
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
	auth    *auth.Authorization
	sinks   []output.Sink
}

func NewIngester(ctx context.Context, log logrus.FieldLogger, conf *Config, clockDrift *time.Duration, geoipProvider geoip.Provider, cache store.Cache) (*Ingester, error) {
	a, err := auth.NewAuthorization(log, conf.Authorization)
	if err != nil {
		return nil, err
	}

	e := &Ingester{
		log:     log.WithField("server/module", ServiceType),
		config:  conf,
		auth:    a,
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

	if err := e.auth.Start(ctx); err != nil {
		return fmt.Errorf("failed to start authorization: %w", err)
	}

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

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("failed to get metadata from context")
	}

	var user *auth.User

	var group *auth.Group

	if e.config.Authorization.Enabled {
		authorization := md.Get("authorization")

		if len(authorization) == 0 {
			return nil, status.Error(codes.Unauthenticated, "no authorization header provided")
		}

		username, err := e.auth.IsAuthorized(authorization[0])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "failed to authorize user")
		}

		if username == "" {
			return nil, status.Error(codes.Unauthenticated, "unauthorized")
		}

		user, group, err = e.auth.GetUserAndGroup(username)
		if err != nil && e.config.Authorization.Enabled {
			return nil, status.Error(codes.Unauthenticated, err.Error())
		}
	}

	filteredEvents, err := e.handler.Events(ctx, req.GetEvents(), user, group)
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
