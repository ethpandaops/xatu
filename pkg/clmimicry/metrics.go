package clmimicry

import (
	"github.com/prometheus/client_golang/prometheus"
)

// Metrics provides simplified metrics for the sharding system
type Metrics struct {
	// Keep decoratedEvents for backward compatibility
	decoratedEvents *prometheus.CounterVec

	// Core metrics for event processing
	eventsTotal     *prometheus.CounterVec
	eventsProcessed *prometheus.CounterVec
	eventsFiltered  *prometheus.CounterVec

	// Sharding decision metrics
	shardingDecisions *prometheus.CounterVec
}

// NewMetrics creates a new metrics instance with simplified metrics
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEvents: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "libp2p_decorated_events_total",
				Help:      "Counts number of decorated events when we received them. Neither of the sharding layers have been applied.",
			},
			[]string{"type", "network_id"},
		),
		eventsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "events_total",
				Help:      "Total number of events received by type",
			},
			[]string{"event_type", "network_id"},
		),
		eventsProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "events_processed_total",
				Help:      "Number of events processed after sharding",
			},
			[]string{"event_type", "network_id"},
		),
		eventsFiltered: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "events_filtered_total",
				Help:      "Number of events filtered out by sharding",
			},
			[]string{"event_type", "network_id"},
		),
		shardingDecisions: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sharding_decisions_total",
				Help:      "Sharding decisions made by reason",
			},
			[]string{"event_type", "reason", "network_id"},
		),
	}

	// Register all metrics
	prometheus.MustRegister(m.decoratedEvents)
	prometheus.MustRegister(m.eventsTotal)
	prometheus.MustRegister(m.eventsProcessed)
	prometheus.MustRegister(m.eventsFiltered)
	prometheus.MustRegister(m.shardingDecisions)

	return m
}

// AddEvent records that an event was received
func (m *Metrics) AddEvent(eventType, network string) {
	m.eventsTotal.WithLabelValues(eventType, network).Inc()
}

// AddProcessedMessage records that an event was processed
func (m *Metrics) AddProcessedMessage(eventType, network string) {
	m.eventsProcessed.WithLabelValues(eventType, network).Inc()
}

// AddSkippedMessage records that an event was filtered out
func (m *Metrics) AddSkippedMessage(eventType, network string) {
	m.eventsFiltered.WithLabelValues(eventType, network).Inc()
}

// AddShardingDecision records the reason for a sharding decision
func (m *Metrics) AddShardingDecision(eventType, reason, network string) {
	m.shardingDecisions.WithLabelValues(eventType, reason, network).Inc()
}

// AddDecoratedEvent tracks decorated events (before sharding)
func (m *Metrics) AddDecoratedEvent(count float64, eventType, network string) {
	m.decoratedEvents.WithLabelValues(eventType, network).Add(count)
}
