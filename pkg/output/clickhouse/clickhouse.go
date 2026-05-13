// Package clickhouse implements an output sink that writes DecoratedEvents
// directly to ClickHouse using the shared writer + router stack from
// pkg/clickhouse.
//
// Unlike the other sinks (xatu, kafka, http, stdout), this sink does NOT
// use processor.BatchItemProcessor. Each call to HandleNewDecoratedEvents
// flushes the entire input slice as one columnar INSERT per affected
// table. This preserves cannon's per-epoch atomicity: one deriver
// callback delivers one full epoch, which maps to one CH INSERT per
// table covering exactly that epoch. The deriver only advances its
// coordinator checkpoint after this call returns nil — so checkpoint
// progress is gated on CH ack, not on a queued-for-batching ack.
//
// As a consequence, the output.Config.ShippingMethod field is ignored
// for this sink type. A non-Sync setting logs a warning at construction.
package clickhouse

import (
	"context"
	"errors"
	"fmt"
	"strings"

	chwriter "github.com/ethpandaops/xatu/pkg/clickhouse"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route/all"
	chrouter "github.com/ethpandaops/xatu/pkg/clickhouse/router"
	"github.com/ethpandaops/xatu/pkg/clickhouse/telemetry"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// SinkType identifies this sink in output.Config.
const SinkType = "clickhouse"

var _ outputSink = (*Sink)(nil)

// outputSink mirrors output.Sink's contract so we can assert compile-time
// compliance without importing pkg/output (which would cause a cycle).
type outputSink interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	Type() string
	Name() string
	HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error
	HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error
}

// chWriter is the subset of chwriter.Writer the sink uses at runtime
// (RegisterBatchFactories is called once at construction on the concrete
// type, so it lives outside this interface). Defined as an interface so
// tests can substitute a stub without spinning up a real CH connection.
type chWriter interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	FlushTableEvents(
		ctx context.Context,
		tableEvents map[string][]*xatu.DecoratedEvent,
	) *chwriter.FlushResult
}

// Sink writes DecoratedEvents straight to ClickHouse, bypassing the
// xatu server / Kafka path.
type Sink struct {
	name             string
	log              logrus.FieldLogger
	writer           chWriter
	router           *chrouter.Engine
	filter           xatu.EventFilter
	restrictPrefixes []string
}

// New constructs a clickhouse sink. shippingMethod is accepted for
// interface uniformity but ignored — see the package-level godoc.
func New(
	name string,
	config *Config,
	log logrus.FieldLogger,
	filterConfig *xatu.EventFilterConfig,
	shippingMethod processor.ShippingMethod,
) (*Sink, error) {
	if config == nil {
		return nil, errors.New("config is required")
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	sLog := log.
		WithField("output_name", name).
		WithField("output_type", SinkType)

	if shippingMethod != "" && shippingMethod != processor.ShippingMethodSync {
		sLog.WithField("shipping_method", shippingMethod).
			Warn("clickhouse sink ignores shippingMethod — flushes are inline per call to preserve per-batch atomicity")
	}

	metrics := telemetry.NewMetrics("xatu", config.MetricsSubsystem)

	writer, err := chwriter.NewWriter(sLog, &config.Config, metrics)
	if err != nil {
		return nil, fmt.Errorf("creating clickhouse writer: %w", err)
	}

	routes, err := all.All()
	if err != nil {
		return nil, fmt.Errorf("registering clickhouse routes: %w", err)
	}

	if len(config.RestrictToTablePrefixes) > 0 {
		routes = filterRoutesByTablePrefix(routes, config.RestrictToTablePrefixes)
		if len(routes) == 0 {
			return nil, fmt.Errorf(
				"restrictToTablePrefixes %v matched no routes",
				config.RestrictToTablePrefixes,
			)
		}

		sLog.WithField("count", len(routes)).
			WithField("prefixes", config.RestrictToTablePrefixes).
			Info("Restricted clickhouse route catalog by table prefix")
	}

	router := chrouter.New(sLog, routes, nil, nil, metrics)

	writer.RegisterBatchFactories(routes)

	filter, err := xatu.NewEventFilter(filterConfig)
	if err != nil {
		return nil, err
	}

	return &Sink{
		name:             name,
		log:              sLog,
		writer:           writer,
		router:           router,
		filter:           filter,
		restrictPrefixes: append([]string(nil), config.RestrictToTablePrefixes...),
	}, nil
}

// Name returns the sink instance's user-supplied name.
func (s *Sink) Name() string {
	return s.name
}

// Type returns the sink type identifier.
func (s *Sink) Type() string {
	return SinkType
}

// Start dials the underlying ClickHouse pool and validates table presence.
func (s *Sink) Start(ctx context.Context) error {
	return s.writer.Start(ctx)
}

// Stop closes the ClickHouse pool.
func (s *Sink) Stop(ctx context.Context) error {
	return s.writer.Stop(ctx)
}

// HandleNewDecoratedEvent flushes a single event.
func (s *Sink) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	return s.HandleNewDecoratedEvents(ctx, []*xatu.DecoratedEvent{event})
}

// HandleNewDecoratedEvents routes the entire input slice and flushes one
// columnar INSERT per affected table. Returns the first table-flush error
// encountered (joined) so the caller's checkpoint does not advance on
// partial failure.
func (s *Sink) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	if len(events) == 0 {
		return nil
	}

	tableEvents := make(map[string][]*xatu.DecoratedEvent, 8)

	for _, event := range events {
		if event == nil {
			continue
		}

		shouldDrop, err := s.filter.ShouldBeDropped(event)
		if err != nil {
			return fmt.Errorf("filtering event: %w", err)
		}

		if shouldDrop {
			continue
		}

		outcome := s.router.Route(event)

		switch outcome.Status {
		case chrouter.StatusDelivered:
			for _, r := range outcome.Results {
				tableEvents[r.Table] = append(tableEvents[r.Table], event)
			}
		case chrouter.StatusErrored:
			if len(s.restrictPrefixes) > 0 {
				return fmt.Errorf(
					"no route registered for event %s (route catalog restricted to prefixes %v) — refusing to advance",
					event.GetEvent().GetName(),
					s.restrictPrefixes,
				)
			}

			return fmt.Errorf(
				"no route registered for event %s — refusing to advance",
				event.GetEvent().GetName(),
			)
		case chrouter.StatusRejected:
			// Permanently invalid (e.g. nil event). Router has already
			// metric'd it; drop silently.
		}
	}

	if len(tableEvents) == 0 {
		return nil
	}

	result := s.writer.FlushTableEvents(ctx, tableEvents)

	if len(result.InvalidEvents) > 0 {
		s.log.WithField("count", len(result.InvalidEvents)).
			Warn("clickhouse flush produced invalid events that could not be flattened — dropping")
	}

	if err := result.Err(); err != nil {
		return fmt.Errorf("clickhouse flush: %w", err)
	}

	return nil
}

func filterRoutesByTablePrefix(routes []route.Route, prefixes []string) []route.Route {
	out := make([]route.Route, 0, len(routes))

	for _, r := range routes {
		table := r.TableName()
		for _, p := range prefixes {
			if strings.HasPrefix(table, p) {
				out = append(out, r)

				break
			}
		}
	}

	return out
}
