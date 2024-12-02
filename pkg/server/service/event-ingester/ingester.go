package eventingester

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	ocodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	log = log.WithField("server/module", ServiceType)

	a, err := auth.NewAuthorization(log, conf.Authorization)
	if err != nil {
		return nil, err
	}

	e := &Ingester{
		log:     log,
		config:  conf,
		auth:    a,
		handler: NewHandler(log, clockDrift, geoipProvider, cache, conf.ClientNameSalt),
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
	ctx, span := observability.Tracer().Start(ctx,
		"EventIngester.CreateEvents",
		trace.WithAttributes(attribute.Int64("events", int64(len(req.GetEvents())))),
	)
	defer span.End()

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
			errMsg := errors.New("no authorization header provided")

			span.SetStatus(ocodes.Error, errMsg.Error())

			return nil, status.Error(codes.Unauthenticated, errMsg.Error())
		}

		username, err := e.auth.IsAuthorized(authorization[0])
		if err != nil {
			errMsg := fmt.Errorf("failed to authorize user: %w", err)

			span.SetStatus(ocodes.Error, errMsg.Error())

			return nil, status.Error(codes.Unauthenticated, errMsg.Error())
		}

		if username == "" {
			errMsg := errors.New("unauthorized")

			span.SetStatus(ocodes.Error, errMsg.Error())

			return nil, status.Error(codes.Unauthenticated, errMsg.Error())
		}

		user, group, err = e.auth.GetUserAndGroup(username)
		if err != nil && e.config.Authorization.Enabled {
			errMsg := fmt.Errorf("failed to get user and group: %w", err)

			span.SetStatus(ocodes.Error, errMsg.Error())

			return nil, status.Error(codes.Unauthenticated, errMsg.Error())
		}

		span.SetAttributes(attribute.String("user", username))
		span.SetAttributes(attribute.String("group", group.Name()))
	}

	filteredEvents, err := e.handler.Events(ctx, req.GetEvents(), user, group)
	if err != nil {
		errMsg := fmt.Errorf("failed to filter events: %w", err)

		span.SetStatus(ocodes.Error, errMsg.Error())

		return nil, status.Error(codes.Internal, errMsg.Error())
	}

	for _, sink := range e.sinks {
		_, span := observability.Tracer().Start(ctx,
			"EventIngester.CreateEvents.SendEventsToSink",
			trace.WithAttributes(
				attribute.String("sink", sink.Name()),
				attribute.Int64("events", int64(len(filteredEvents))),
			),
		)

		if err := sink.HandleNewDecoratedEvents(ctx, filteredEvents); err != nil {
			errMsg := fmt.Errorf("failed to handle new decorated events: %w", err)

			span.SetStatus(ocodes.Error, errMsg.Error())
			span.End()

			return nil, status.Error(codes.Internal, errMsg.Error())
		}

		span.End()
	}

	return &xatu.CreateEventsResponse{
		EventsIngested: &wrapperspb.UInt64Value{
			Value: uint64(len(filteredEvents)),
		},
	}, nil
}

func (e *Ingester) CreateSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(e.config.Outputs))

	for i, out := range e.config.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodAsync

			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			e.log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}
