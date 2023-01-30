package sentry

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events received",
		}, []string{"type", "network"}),
	}

	prometheus.MustRegister(m.decoratedEventTotal)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType, network string) {
	m.decoratedEventTotal.WithLabelValues(eventType, network).Add(float64(count))
}
