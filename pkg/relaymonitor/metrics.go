package relaymonitor

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec
	networkName         string
}

func NewMetrics(namespace, networkName string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events received",
		}, []string{"type"}),
		networkName: networkName,
	}

	prometheus.MustRegister(
		m.decoratedEventTotal,
	)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType string) {
	m.decoratedEventTotal.WithLabelValues(eventType).Add(float64(count))
}
