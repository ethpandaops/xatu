package eventingester

import (
	"context"
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
)

// Pipeline manages the shared event processing infrastructure used by
// different transport layers (gRPC, HTTP) to ingest events.
type Pipeline struct {
	log     logrus.FieldLogger
	config  *Config
	auth    *auth.Authorization
	handler *Handler
	sinks   []output.Sink
}

// NewPipeline creates a new event processing pipeline.
func NewPipeline(
	ctx context.Context,
	log logrus.FieldLogger,
	config *Config,
	clockDrift *time.Duration,
	geoipProvider geoip.Provider,
	cache store.Cache,
) (*Pipeline, error) {
	a, err := auth.NewAuthorization(log, config.Authorization)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization: %w", err)
	}

	p := &Pipeline{
		log:     log.WithField("component", "pipeline"),
		config:  config,
		auth:    a,
		handler: NewHandler(log, clockDrift, geoipProvider, cache, config.ClientNameSalt),
	}

	sinks, err := p.createSinks()
	if err != nil {
		return nil, fmt.Errorf("failed to create sinks: %w", err)
	}

	p.sinks = sinks

	return p, nil
}

// Start starts the pipeline components (auth and sinks).
func (p *Pipeline) Start(ctx context.Context) error {
	if err := p.auth.Start(ctx); err != nil {
		return fmt.Errorf("failed to start authorization: %w", err)
	}

	for _, sink := range p.sinks {
		if err := sink.Start(ctx); err != nil {
			return fmt.Errorf("failed to start sink %s: %w", sink.Name(), err)
		}
	}

	return nil
}

// Stop stops all sinks.
func (p *Pipeline) Stop(ctx context.Context) error {
	for _, sink := range p.sinks {
		if err := sink.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop sink %s: %w", sink.Name(), err)
		}
	}

	return nil
}

// Handler returns the event handler.
func (p *Pipeline) Handler() *Handler {
	return p.handler
}

// Auth returns the authorization component.
func (p *Pipeline) Auth() *auth.Authorization {
	return p.auth
}

// AuthEnabled returns whether authorization is enabled.
func (p *Pipeline) AuthEnabled() bool {
	return p.config.Authorization.Enabled
}

// ProcessAndSend processes events through the handler and sends to all sinks.
// Returns the number of events that were processed and sent.
func (p *Pipeline) ProcessAndSend(
	ctx context.Context,
	events []*xatu.DecoratedEvent,
	user *auth.User,
	group *auth.Group,
	spanPrefix string,
) (uint64, error) {
	filteredEvents, err := p.handler.Events(ctx, events, user, group)
	if err != nil {
		return 0, fmt.Errorf("failed to process events: %w", err)
	}

	for _, sink := range p.sinks {
		_, span := observability.Tracer().Start(ctx,
			spanPrefix+".SendEventsToSink",
			trace.WithAttributes(
				attribute.String("sink", sink.Name()),
				attribute.Int64("events", int64(len(filteredEvents))),
			),
		)

		if err := sink.HandleNewDecoratedEvents(ctx, filteredEvents); err != nil {
			span.SetStatus(ocodes.Error, err.Error())
			span.End()

			return 0, fmt.Errorf("failed to send events to sink %s: %w", sink.Name(), err)
		}

		span.End()
	}

	return uint64(len(filteredEvents)), nil
}

// createSinks creates output sinks from configuration.
func (p *Pipeline) createSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(p.config.Outputs))

	for i, out := range p.config.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodSync
			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(
			out.Name,
			out.SinkType,
			out.Config,
			p.log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create sink %s: %w", out.Name, err)
		}

		sinks[i] = sink
	}

	return sinks, nil
}
