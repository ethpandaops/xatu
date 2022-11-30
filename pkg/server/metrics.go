package server

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventsTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_events_received_total",
			Help:      "Total number of decorated events received",
		}, []string{"event", "sentry_id"}),
	}

	prometheus.MustRegister(m.decoratedEventsTotal)

	return m
}

func (m *Metrics) AddDecoratedEventReceived(count int, event string, sentryID string) {
	m.decoratedEventsTotal.WithLabelValues(event, sentryID).Add(float64(count))
}
