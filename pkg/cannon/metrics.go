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
		}, []string{"type", "network"}),
		deriverLocation: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "deriver_location",
			Help:      "Location of the cannon event deriver",
		}, []string{"type", "network"}),
	}

	prometheus.MustRegister(m.decoratedEventTotal)
	prometheus.MustRegister(m.deriverLocation)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType *xatu.DecoratedEvent, network string) {
	m.decoratedEventTotal.WithLabelValues(eventType.Event.Name.String(), network).Add(float64(count))
}

func (m *Metrics) SetDeriverLocation(location uint64, eventType xatu.CannonType, network string) {
	m.deriverLocation.WithLabelValues(eventType.String(), network).Set(float64(location))
}
