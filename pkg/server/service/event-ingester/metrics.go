package eventingester

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventsTotal         *prometheus.CounterVec
	decoratedEventsFromUserTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_events_received_total",
			Help:      "Total number of decorated events received",
		}, []string{"event", "group"}),
		decoratedEventsFromUserTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_events_from_user_total",
			Help:      "Total number of decorated events from user",
		}, []string{"user", "group"}),
	}

	prometheus.MustRegister(m.decoratedEventsTotal)

	return m
}

func (m *Metrics) AddDecoratedEventReceived(count int, event, sentryID string) {
	m.decoratedEventsTotal.WithLabelValues(event, sentryID).Add(float64(count))
}

func (m *Metrics) AddDecoratedEventFromUserReceived(count int, user, group string) {
	m.decoratedEventsFromUserTotal.WithLabelValues(user, group).Add(float64(count))
}
