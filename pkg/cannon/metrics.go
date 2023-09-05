package cannon

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec
	location            *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events created by the cannon",
		}, []string{"type"}),
		location: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "location",
			Help:      "Location of the cannon",
		}, []string{"type"}),
	}

	prometheus.MustRegister(m.decoratedEventTotal)
	prometheus.MustRegister(m.location)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType xatu.CannonType) {
	m.decoratedEventTotal.WithLabelValues(eventType.String()).Add(float64(count))
}

func (m *Metrics) SetLocation(location uint64, eventType xatu.CannonType) {
	m.location.WithLabelValues(eventType.String()).Set(float64(location))
}
