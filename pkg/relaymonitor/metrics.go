package relaymonitor

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec
	relayHealthy        *prometheus.GaugeVec
	networkName         string
}

func NewMetrics(namespace string, networkName string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events received",
		}, []string{"type"}),
		relayHealthy: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "relay_healthy",
			Help:      "Health of the relay",
		}, []string{"relay"}),
		networkName: networkName,
	}

	prometheus.MustRegister(
		m.decoratedEventTotal,
		m.relayHealthy,
	)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType string) {
	m.decoratedEventTotal.WithLabelValues(eventType).Add(float64(count))
}

func (m *Metrics) SetRelayHealth(health float64, relay string) {
	m.relayHealthy.WithLabelValues(relay).Set(health)
}
