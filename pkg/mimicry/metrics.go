package mimicry

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
		}, []string{"type", "network_id"}),
	}

	prometheus.MustRegister(m.decoratedEventTotal)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType, networkID string) {
	m.decoratedEventTotal.WithLabelValues(eventType, networkID).Add(float64(count))
}
