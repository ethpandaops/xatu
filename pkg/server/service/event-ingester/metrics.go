package eventingester

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	decoratedEventsTotal         *prometheus.CounterVec
	decoratedEventsFromUserTotal *prometheus.CounterVec
}

var (
	metricsInstance *Metrics
	metricsOnce     sync.Once
)

func NewMetrics(namespace string) *Metrics {
	metricsOnce.Do(func() {
		metricsInstance = &Metrics{
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

		prometheus.MustRegister(metricsInstance.decoratedEventsTotal)
		prometheus.MustRegister(metricsInstance.decoratedEventsFromUserTotal)
	})

	return metricsInstance
}

func (m *Metrics) AddDecoratedEventReceived(count int, event, sentryID string) {
	m.decoratedEventsTotal.WithLabelValues(event, sentryID).Add(float64(count))
}

func (m *Metrics) AddDecoratedEventFromUserReceived(count int, user, group string) {
	m.decoratedEventsFromUserTotal.WithLabelValues(user, group).Add(float64(count))
}
