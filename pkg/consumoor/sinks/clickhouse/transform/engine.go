package transform

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/consumoor/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// Result holds the output of routing a single event: the target
// table name and the flat rows to insert.
type Result struct {
	Table string
	Rows  []map[string]any
}

// Outcome holds routed rows and the overall delivery status for one
// event. A non-delivered status means rows should not be written.
type Outcome struct {
	Results []Result
	Status  Status
}

// Engine maps event names to registered routes and dispatches incoming
// events to the appropriate route handlers.
type Engine struct {
	log logrus.FieldLogger

	// routesByEvent maps event names to the list of routes that handle
	// that event. Most events have one route, but conditional routing
	// produces multiple.
	routesByEvent map[xatu.Event_Name][]flattener.Route

	metrics *telemetry.Metrics
}

// New creates a routing engine with the given routes.
func New(
	log logrus.FieldLogger,
	routes []flattener.Route,
	disabledEvents []xatu.Event_Name,
	metrics *telemetry.Metrics,
) *Engine {
	r := &Engine{
		log:           log.WithField("component", "router"),
		routesByEvent: make(map[xatu.Event_Name][]flattener.Route, len(routes)),
		metrics:       metrics,
	}

	disabled := make(map[xatu.Event_Name]struct{}, len(disabledEvents))
	for _, name := range disabledEvents {
		disabled[name] = struct{}{}
	}

	// Register routes by event name.
	for _, route := range routes {
		for _, name := range route.EventNames() {
			if _, isDisabled := disabled[name]; isDisabled {
				continue
			}

			r.routesByEvent[name] = append(r.routesByEvent[name], route)
		}
	}

	// Log registration summary
	log.WithField("registered_events", len(r.routesByEvent)).
		WithField("disabled_events", len(disabled)).
		Info("Routing engine initialized")

	return r
}

// Route processes a single DecoratedEvent through the routing pipeline:
// extract metadata, find matching routes, flatten the event, and
// return the results grouped by target table.
func (r *Engine) Route(event *xatu.DecoratedEvent) Outcome {
	if event == nil || event.GetEvent() == nil {
		return Outcome{Status: StatusRejected}
	}

	eventName := event.GetEvent().GetName()

	// Look up routes for this event.
	routesForEvent, ok := r.routesByEvent[eventName]
	if !ok {
		r.metrics.MessagesDropped().WithLabelValues(eventName.String(), "no_flattener").Inc()

		return Outcome{Status: StatusDelivered}
	}

	// Extract shared metadata once
	meta := metadata.Extract(event)

	results := make([]Result, 0, len(routesForEvent))

	for _, route := range routesForEvent {
		// Check conditional routing
		if !route.ShouldProcess(event) {
			r.metrics.MessagesDropped().WithLabelValues(eventName.String(), "filtered").Inc()

			continue
		}

		rows, err := route.Flatten(event, meta)
		if err != nil {
			r.log.WithError(err).
				WithField("event_name", eventName.String()).
				WithField("table", route.TableName()).
				Warn("Route error")

			r.metrics.FlattenErrors().WithLabelValues(eventName.String(), route.TableName()).Inc()

			return Outcome{Status: StatusRejected}
		}

		if len(rows) == 0 {
			continue
		}

		results = append(results, Result{
			Table: route.TableName(),
			Rows:  rows,
		})
	}

	for _, result := range results {
		r.metrics.MessagesRouted().WithLabelValues(eventName.String(), result.Table).Inc()
	}

	return Outcome{
		Results: results,
		Status:  StatusDelivered,
	}
}
