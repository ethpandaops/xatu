package router

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	"github.com/ethpandaops/xatu/pkg/clickhouse/telemetry"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const logSampleInterval = 30 * time.Second

// Result holds the routing decision for a single event: the target
// table name and the route that will handle flattening.
type Result struct {
	Table string
	Route route.Route
}

// Outcome holds routing decisions and the overall delivery status for
// one event. A non-delivered status means the event should not be written.
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
	routesByEvent map[xatu.Event_Name][]route.Route

	// disabledByConfig marks event names that had at least one registered
	// route before config-driven filtering removed all of them. These are
	// dropped (StatusDelivered) rather than NAK'd at Route() time.
	disabledByConfig map[xatu.Event_Name]struct{}

	metrics    *telemetry.Metrics
	logSampler *telemetry.LogSampler
}

// New creates a routing engine with the given routes.
func New(
	log logrus.FieldLogger,
	routes []route.Route,
	disabledEvents []xatu.Event_Name,
	disabledTables map[string]struct{},
	metrics *telemetry.Metrics,
) *Engine {
	r := &Engine{
		log:              log.WithField("component", "router"),
		routesByEvent:    make(map[xatu.Event_Name][]route.Route, len(routes)),
		disabledByConfig: make(map[xatu.Event_Name]struct{}, len(disabledEvents)),
		metrics:          metrics,
		logSampler:       telemetry.NewLogSampler(logSampleInterval),
	}

	disabled := make(map[xatu.Event_Name]struct{}, len(disabledEvents))
	for _, name := range disabledEvents {
		disabled[name] = struct{}{}
	}

	// Register routes by event name, skipping routes whose target table is
	// disabled or whose event is disabled. Event names that had any route
	// before filtering are recorded in disabledByConfig so they are dropped
	// (not NAK'd) at Route() time when all their routes are filtered out.
	for _, route := range routes {
		tableDisabled := false
		if _, ok := disabledTables[route.TableName()]; ok {
			tableDisabled = true
		}

		for _, name := range route.EventNames() {
			_, eventDisabled := disabled[name]

			if tableDisabled || eventDisabled {
				r.disabledByConfig[name] = struct{}{}

				continue
			}

			r.routesByEvent[name] = append(r.routesByEvent[name], route)
		}
	}

	// An event that had at least one surviving route is not disabled —
	// drop it from disabledByConfig so Route() does not short-circuit it.
	for name := range r.routesByEvent {
		delete(r.disabledByConfig, name)
	}

	// Log registration summary
	log.WithField("registered_events", len(r.routesByEvent)).
		WithField("disabled_events", len(disabled)).
		WithField("disabled_tables", len(disabledTables)).
		Info("Routing engine initialized")

	return r
}

// Route processes a single DecoratedEvent through the routing pipeline:
// extract metadata, find matching routes, and return routing decisions.
func (r *Engine) Route(event *xatu.DecoratedEvent) Outcome {
	if event == nil || event.GetEvent() == nil {
		return Outcome{Status: StatusRejected}
	}

	eventName := event.GetEvent().GetName()

	// Look up routes for this event. Intentionally unsupported events
	// are dropped (status delivered, no rows). Unknown/unexpected event
	// types are NAK'd (StatusErrored) so Kafka does not advance offsets,
	// preventing silent data loss when new event types appear before a
	// matching route is deployed.
	routesForEvent, ok := r.routesByEvent[eventName]
	if !ok {
		if _, disabledByConfig := r.disabledByConfig[eventName]; disabledByConfig {
			if r.metrics != nil {
				r.metrics.MessagesDropped().WithLabelValues(eventName.String(), "disabled_by_config").Inc()
			}

			if ok, suppressed := r.logSampler.Allow("config_drop:" + eventName.String()); ok {
				entry := r.log.WithField("event_name", eventName.String())
				if suppressed > 0 {
					entry = entry.WithField("suppressed", suppressed)
				}

				entry.Debug("Event has no enabled routes after config filtering — dropping")
			}

			return Outcome{Status: StatusDelivered}
		}

		if reason, intentionallyUnsupported := route.UnsupportedReason(eventName); intentionallyUnsupported {
			if r.metrics != nil {
				r.metrics.MessagesDropped().WithLabelValues(eventName.String(), "no_flattener").Inc()
			}

			if ok, suppressed := r.logSampler.Allow("drop:" + eventName.String()); ok {
				entry := r.log.
					WithField("event_name", eventName.String()).
					WithField("reason", reason)
				if suppressed > 0 {
					entry = entry.WithField("suppressed", suppressed)
				}

				entry.Debug("No route registered for intentionally unsupported event — dropping")
			}

			return Outcome{Status: StatusDelivered}
		}

		if r.metrics != nil {
			r.metrics.MessagesDropped().WithLabelValues(eventName.String(), "no_route_nack").Inc()
		}

		if ok, suppressed := r.logSampler.Allow(eventName.String()); ok {
			entry := r.log.WithField("event_name", eventName.String())
			if suppressed > 0 {
				entry = entry.WithField("suppressed", suppressed)
			}

			entry.Warn("No route registered for event — messages will be NAK'd until a matching route is deployed")
		}

		return Outcome{Status: StatusErrored}
	}

	results := make([]Result, 0, len(routesForEvent))

	for _, route := range routesForEvent {
		// Check conditional routing
		if !route.ShouldProcess(event) {
			if r.metrics != nil {
				r.metrics.MessagesDropped().WithLabelValues(eventName.String(), "filtered").Inc()
			}

			continue
		}

		results = append(results, Result{
			Table: route.TableName(),
			Route: route,
		})
	}

	if r.metrics != nil {
		for _, result := range results {
			r.metrics.MessagesRouted().WithLabelValues(eventName.String(), result.Table).Inc()
		}

		if ts := event.GetEvent().GetDateTime(); ts != nil {
			lag := time.Since(ts.AsTime()).Seconds()
			if lag >= 0 {
				r.metrics.EventLag().WithLabelValues(eventName.String()).Observe(lag)
			}
		}
	}

	return Outcome{
		Results: results,
		Status:  StatusDelivered,
	}
}
