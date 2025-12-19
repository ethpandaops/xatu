package ethstats

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics contains Prometheus metrics for the ethstats service.
type Metrics struct {
	connectionsTotal  *prometheus.CounterVec
	connectionsActive prometheus.Gauge
	eventsReceived    *prometheus.CounterVec
	eventsExported    *prometheus.CounterVec
	eventsFiltered    *prometheus.CounterVec
	eventErrors       *prometheus.CounterVec
	messageLatency    *prometheus.HistogramVec
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		connectionsTotal: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "connections_total",
			Help:      "Total number of WebSocket connections",
		}, []string{"status"}),
		connectionsActive: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connections_active",
			Help:      "Number of active WebSocket connections",
		}),
		eventsReceived: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_received_total",
			Help:      "Total number of events received from ethstats clients",
		}, []string{"event_type"}),
		eventsExported: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_exported_total",
			Help:      "Total number of events exported to sinks",
		}, []string{"event_type", "sink"}),
		eventsFiltered: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "events_filtered_total",
			Help:      "Total number of events filtered out by user/group event filters",
		}, []string{"event_type"}),
		eventErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "event_errors_total",
			Help:      "Total number of event processing errors",
		}, []string{"event_type", "error_type"}),
		messageLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_processing_seconds",
			Help:      "Histogram of message processing latency in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"message_type"}),
	}

	return m
}

// IncConnectionsTotal increments the total connections counter.
func (m *Metrics) IncConnectionsTotal(status string) {
	m.connectionsTotal.WithLabelValues(status).Inc()
}

// IncConnectionsActive increments the active connections gauge.
func (m *Metrics) IncConnectionsActive() {
	m.connectionsActive.Inc()
}

// DecConnectionsActive decrements the active connections gauge.
func (m *Metrics) DecConnectionsActive() {
	m.connectionsActive.Dec()
}

// IncEventsReceived increments the events received counter.
func (m *Metrics) IncEventsReceived(eventType string) {
	m.eventsReceived.WithLabelValues(eventType).Inc()
}

// IncEventsExported increments the events exported counter.
func (m *Metrics) IncEventsExported(eventType, sink string) {
	m.eventsExported.WithLabelValues(eventType, sink).Inc()
}

// IncEventsFiltered increments the events filtered counter.
func (m *Metrics) IncEventsFiltered(eventType string) {
	m.eventsFiltered.WithLabelValues(eventType).Inc()
}

// IncEventErrors increments the event errors counter.
func (m *Metrics) IncEventErrors(eventType, errorType string) {
	m.eventErrors.WithLabelValues(eventType, errorType).Inc()
}

// ObserveMessageLatency records message processing latency.
func (m *Metrics) ObserveMessageLatency(messageType string, seconds float64) {
	m.messageLatency.WithLabelValues(messageType).Observe(seconds)
}
