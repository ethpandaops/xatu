package eventingester

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
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
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceType = "event-ingester"
)

type Ingester struct {
	xatu.UnimplementedEventIngesterServer

	log      logrus.FieldLogger
	config   *Config
	pipeline *Pipeline

	healthServer *health.Server
}

func NewIngester(
	ctx context.Context,
	log logrus.FieldLogger,
	conf *Config,
	clockDrift *time.Duration,
	geoipProvider geoip.Provider,
	cache store.Cache,
	healthServer *health.Server,
) (*Ingester, error) {
	log = log.WithField("server/module", ServiceType)

	pipeline, err := NewPipeline(ctx, log, conf, clockDrift, geoipProvider, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	e := &Ingester{
		log:          log,
		config:       conf,
		pipeline:     pipeline,
		healthServer: healthServer,
	}

	return e, nil
}

// Name returns the name of this service
func (e *Ingester) Name() string {
	return "event-ingester"
}

func (e *Ingester) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("Starting module")

	if err := e.pipeline.Start(ctx); err != nil {
		return fmt.Errorf("failed to start pipeline: %w", err)
	}

	xatu.RegisterEventIngesterServer(grpcServer, e)

	e.healthServer.SetServingStatus(e.Name(), grpc_health_v1.HealthCheckResponse_SERVING)

	return nil
}

func (e *Ingester) Stop(ctx context.Context) error {
	e.log.Info("Stopping module")

	e.healthServer.SetServingStatus(e.Name(), grpc_health_v1.HealthCheckResponse_NOT_SERVING)

	if err := e.pipeline.Stop(ctx); err != nil {
		return status.Error(codes.Internal, err.Error())
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

	var (
		user  *auth.User
		group *auth.Group
	)

	if e.pipeline.AuthEnabled() {
		authorization := md.Get("authorization")

		if len(authorization) == 0 {
			errMsg := errors.New("no authorization header provided")

			span.SetStatus(ocodes.Error, errMsg.Error())

			return nil, status.Error(codes.Unauthenticated, errMsg.Error())
		}

		username, err := e.pipeline.Auth().IsAuthorized(authorization[0])
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

		user, group, err = e.pipeline.Auth().GetUserAndGroup(username)
		if err != nil {
			errMsg := fmt.Errorf("failed to get user and group: %w", err)

			span.SetStatus(ocodes.Error, errMsg.Error())

			return nil, status.Error(codes.Unauthenticated, errMsg.Error())
		}

		span.SetAttributes(attribute.String("user", username))
		span.SetAttributes(attribute.String("group", group.Name()))
	}

	filteredCount, err := e.pipeline.ProcessAndSend(ctx, req.GetEvents(), user, group, "EventIngester.CreateEvents")
	if err != nil {
		span.SetStatus(ocodes.Error, err.Error())

		return nil, status.Error(codes.Internal, err.Error())
	}

	return &xatu.CreateEventsResponse{
		EventsIngested: &wrapperspb.UInt64Value{
			Value: filteredCount,
		},
	}, nil
}
