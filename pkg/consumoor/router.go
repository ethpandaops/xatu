package consumoor

import (
	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

// RouterResult holds the output of routing a single event: the target
// table name and the flat rows to insert.
type RouterResult struct {
	Table string
	Rows  []map[string]any
}

// RouterOutcome holds routed rows and the overall delivery status for one
// event. A non-delivered status means rows should not be written.
type RouterOutcome struct {
	Results []RouterResult
	Status  DeliveryStatus
}

// Router maps event names to registered routes and dispatches incoming
// events to the appropriate route handlers.
type Router struct {
	log logrus.FieldLogger

	// routesByEvent maps event names to the list of routes that handle
	// that event. Most events have one route, but conditional routing
	// produces multiple.
	routesByEvent map[xatu.Event_Name][]flattener.Route

	// disabledEvents is a set of event names that should be silently
	// dropped without processing.
	disabledEvents map[xatu.Event_Name]struct{}

	metrics *Metrics
}

// NewRouter creates a Router with the given routes and disabled events.
func NewRouter(
	log logrus.FieldLogger,
	routes []flattener.Route,
	disabledEvents []xatu.Event_Name,
	metrics *Metrics,
) *Router {
	r := &Router{
		log:            log.WithField("component", "router"),
		routesByEvent:  make(map[xatu.Event_Name][]flattener.Route, len(routes)),
		disabledEvents: make(map[xatu.Event_Name]struct{}, len(disabledEvents)),
		metrics:        metrics,
	}

	// Register routes by event name.
	for _, route := range routes {
		for _, name := range route.EventNames() {
			r.routesByEvent[name] = append(r.routesByEvent[name], route)
		}
	}

	// Build disabled events set
	for _, name := range disabledEvents {
		r.disabledEvents[name] = struct{}{}
	}

	// Log registration summary
	log.WithField("registered_events", len(r.routesByEvent)).
		WithField("disabled_events", len(r.disabledEvents)).
		Info("Router initialized")

	return r
}

// Route processes a single DecoratedEvent through the routing pipeline:
// extract metadata, find matching routes, flatten the event, and
// return the results grouped by target table.
func (r *Router) Route(event *xatu.DecoratedEvent) RouterOutcome {
	if event == nil || event.GetEvent() == nil {
		return RouterOutcome{Status: DeliveryStatusRejected}
	}

	eventName := event.GetEvent().GetName()

	// Check if this event is disabled
	if _, disabled := r.disabledEvents[eventName]; disabled {
		r.metrics.messagesDropped.WithLabelValues(eventName.String(), "disabled").Inc()

		return RouterOutcome{Status: DeliveryStatusDelivered}
	}

	// Look up routes for this event.
	routesForEvent, ok := r.routesByEvent[eventName]
	if !ok {
		r.metrics.messagesDropped.WithLabelValues(eventName.String(), "no_flattener").Inc()

		return RouterOutcome{Status: DeliveryStatusDelivered}
	}

	// Extract shared metadata once
	meta := metadata.Extract(event)

	results := make([]RouterResult, 0, len(routesForEvent))

	for _, route := range routesForEvent {
		// Check conditional routing
		if !route.ShouldProcess(event) {
			r.metrics.messagesDropped.WithLabelValues(eventName.String(), "filtered").Inc()

			continue
		}

		rows, err := route.Flatten(event, meta)
		if err != nil {
			r.log.WithError(err).
				WithField("event_name", eventName.String()).
				WithField("table", route.TableName()).
				Warn("Route error")

			r.metrics.flattenErrors.WithLabelValues(eventName.String(), route.TableName()).Inc()

			return RouterOutcome{Status: DeliveryStatusRejected}
		}

		if len(rows) == 0 {
			continue
		}

		results = append(results, RouterResult{
			Table: route.TableName(),
			Rows:  rows,
		})
	}

	for _, result := range results {
		r.metrics.messagesRouted.WithLabelValues(eventName.String(), result.Table).Inc()
	}

	return RouterOutcome{
		Results: results,
		Status:  DeliveryStatusDelivered,
	}
}
