package eventingester

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const (
	ServiceType = "event-ingester"
)

type EventIngester struct {
	xatu.UnimplementedEventIngesterServer

	log    logrus.FieldLogger
	config *Config

	sinks []output.Sink

	metrics *Metrics
}

func New(ctx context.Context, log logrus.FieldLogger, conf *Config) (*EventIngester, error) {
	e := &EventIngester{
		log:    log.WithField("server/module", ServiceType),
		config: conf,

		metrics: NewMetrics("xatu_event_ingester"),
	}

	sinks, err := e.CreateSinks()
	if err != nil {
		return e, err
	}

	e.sinks = sinks

	return e, nil
}

func (e *EventIngester) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("starting module")

	xatu.RegisterEventIngesterServer(grpcServer, e)

	return nil
}

func (e *EventIngester) Stop(ctx context.Context) error {
	e.log.Info("stopping module")

	return nil
}

func (e *EventIngester) CreateEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	e.log.WithField("events", len(req.Events)).Debug("Received batch of events")

	// TODO(sam.calder-mason): Add clock drift
	receivedAt := timestamppb.New(time.Now())

	p, _ := peer.FromContext(ctx)

	for _, event := range req.Events {
		// TODO(sam.calder-mason): Validate event
		// TODO(sam.calder-mason): Derive client id/name from the request jwt
		e.metrics.AddDecoratedEventReceived(1, event.Meta.Client.Event.Name.String(), "unknown")

		event.Meta.Server = &xatu.ServerMeta{
			Event: &xatu.ServerMeta_Event{
				DateTime: receivedAt,
			},
			Client: &xatu.ServerMeta_Client{
				IP: p.Addr.String(),
			},
		}
	}

	for _, sink := range e.sinks {
		for _, event := range req.Events {
			if err := sink.HandleNewDecoratedEvent(ctx, event); err != nil {
				return nil, err
			}
		}
	}

	return &xatu.CreateEventsResponse{}, nil
}

func (e *EventIngester) CreateSinks() ([]output.Sink, error) {
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
