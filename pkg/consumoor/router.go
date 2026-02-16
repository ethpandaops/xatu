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

// Router maps event names to registered flatteners and dispatches
// incoming events to the appropriate flatteners.
type Router struct {
	log logrus.FieldLogger

	// flatteners maps event names to the list of flatteners that
	// handle that event. Most events have one flattener, but
	// conditional routing produces multiple.
	flatteners map[xatu.Event_Name][]flattener.Flattener

	// disabledEvents is a set of event names that should be silently
	// dropped without processing.
	disabledEvents map[xatu.Event_Name]struct{}

	metrics *Metrics
}

// NewRouter creates a Router with the given flatteners and disabled events.
func NewRouter(
	log logrus.FieldLogger,
	flatteners []flattener.Flattener,
	disabledEvents []xatu.Event_Name,
	metrics *Metrics,
) *Router {
	r := &Router{
		log:            log.WithField("component", "router"),
		flatteners:     make(map[xatu.Event_Name][]flattener.Flattener, len(flatteners)),
		disabledEvents: make(map[xatu.Event_Name]struct{}, len(disabledEvents)),
		metrics:        metrics,
	}

	// Register flatteners by event name
	for _, f := range flatteners {
		for _, name := range f.EventNames() {
			r.flatteners[name] = append(r.flatteners[name], f)
		}
	}

	// Build disabled events set
	for _, name := range disabledEvents {
		r.disabledEvents[name] = struct{}{}
	}

	// Log registration summary
	log.WithField("registered_events", len(r.flatteners)).
		WithField("disabled_events", len(r.disabledEvents)).
		Info("Router initialized")

	return r
}

// Route processes a single DecoratedEvent through the routing pipeline:
// extract metadata, find matching flatteners, flatten the event, and
// return the results grouped by target table.
func (r *Router) Route(event *xatu.DecoratedEvent) []RouterResult {
	if event == nil || event.GetEvent() == nil {
		return nil
	}

	eventName := event.GetEvent().GetName()

	// Check if this event is disabled
	if _, disabled := r.disabledEvents[eventName]; disabled {
		r.metrics.messagesDropped.WithLabelValues(eventName.String(), "disabled").Inc()

		return nil
	}

	// Look up flatteners for this event
	flattenersForEvent, ok := r.flatteners[eventName]
	if !ok {
		r.metrics.messagesDropped.WithLabelValues(eventName.String(), "no_flattener").Inc()

		return nil
	}

	// Extract shared metadata once
	meta := metadata.Extract(event)

	results := make([]RouterResult, 0, len(flattenersForEvent))

	for _, f := range flattenersForEvent {
		// Check conditional routing
		if !f.ShouldProcess(event) {
			r.metrics.messagesDropped.WithLabelValues(eventName.String(), "filtered").Inc()

			continue
		}

		rows, err := f.Flatten(event, meta)
		if err != nil {
			r.log.WithError(err).
				WithField("event_name", eventName.String()).
				WithField("table", f.TableName()).
				Warn("Flattener error")

			r.metrics.flattenErrors.WithLabelValues(eventName.String(), f.TableName()).Inc()

			continue
		}

		if len(rows) == 0 {
			continue
		}

		r.metrics.messagesRouted.WithLabelValues(eventName.String(), f.TableName()).Inc()

		results = append(results, RouterResult{
			Table: f.TableName(),
			Rows:  rows,
		})
	}

	return results
}
