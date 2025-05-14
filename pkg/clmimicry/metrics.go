package clmimicry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// Events metrics.
	decoratedEvents *prometheus.CounterVec

	// Sampling metrics.
	samplingProcessed *prometheus.CounterVec
	samplingSkipped   *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		decoratedEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "decorated_event_total",
				Help:      "The number of decorated events",
			},
			[]string{"event", "network"},
		),
		samplingProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_processed_total",
				Help:      "The number of messages processed after sampling",
			},
			[]string{"event_type"},
		),
		samplingSkipped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_skipped_total",
				Help:      "The number of messages skipped due to sampling",
			},
			[]string{"event_type"},
		),
	}
}

func (m *Metrics) AddDecoratedEvent(count float64, eventType, network string) {
	m.decoratedEvents.WithLabelValues(eventType, network).Add(count)
}

func (m *Metrics) AddProcessedMessage(eventType string) {
	m.samplingProcessed.WithLabelValues(eventType).Inc()
}

func (m *Metrics) AddSkippedMessage(eventType string) {
	m.samplingSkipped.WithLabelValues(eventType).Inc()
}
