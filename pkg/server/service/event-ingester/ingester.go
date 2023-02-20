package eventingester

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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
	e.log.Info("starting module")

	xatu.RegisterEventIngesterServer(grpcServer, e)

	return nil
}

func (e *Ingester) Stop(ctx context.Context) error {
	e.log.Info("stopping module")

	return nil
}

func (e *Ingester) CreateEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	e.log.WithField("events", len(req.Events)).Debug("Received batch of events")

	// TODO(sam.calder-mason): Derive client id/name from the request jwt
	clientID := "unknown"

	filteredEvents, err := e.handler.Events(ctx, clientID, req.Events)
	if err != nil {
		return nil, err
	}

	for _, sink := range e.sinks {
		for _, event := range filteredEvents {
			if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
				return nil, err
			}
		}
	}

	return &xatu.CreateEventsResponse{}, nil
}

func (e *Ingester) CreateSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(e.config.Outputs))

	for i, out := range e.config.Outputs {
		sink, err := output.NewSink(out.SinkType, out.Config, e.log)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}
