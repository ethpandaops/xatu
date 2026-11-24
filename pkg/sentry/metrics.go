package sentry

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec
	executionWatcherUp  *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events received",
		}, []string{"type", "network_id"}),
		executionWatcherUp: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "execution_watcher_up",
			Help:      "Whether an enabled execution watcher is running (1) or still retrying its start (0)",
		}, []string{"watcher"}),
	}

	prometheus.MustRegister(m.decoratedEventTotal, m.executionWatcherUp)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType, networkID string) {
	m.decoratedEventTotal.WithLabelValues(eventType, networkID).Add(float64(count))
}

func (m *Metrics) SetExecutionWatcherUp(watcher string, up bool) {
	value := 0.0
	if up {
		value = 1
	}

	m.executionWatcherUp.WithLabelValues(watcher).Set(value)
}
