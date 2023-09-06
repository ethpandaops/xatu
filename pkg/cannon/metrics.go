package cannon

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec
	deriverLocation     *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events created by the cannon",
		}, []string{"type"}),
		deriverLocation: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "deriver_location",
			Help:      "Location of the cannon event deriver",
		}, []string{"type"}),
	}

	prometheus.MustRegister(m.decoratedEventTotal)
	prometheus.MustRegister(m.deriverLocation)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType xatu.CannonType) {
	m.decoratedEventTotal.WithLabelValues(eventType.String()).Add(float64(count))
}

func (m *Metrics) SetDeriverLocation(location uint64, eventType xatu.CannonType) {
	m.deriverLocation.WithLabelValues(eventType.String()).Set(float64(location))
}
