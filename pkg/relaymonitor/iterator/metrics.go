package iterator

import (
	"github.com/prometheus/client_golang/prometheus"
)

type ConsistencyMetrics struct {
	// Current slot being processed
	CurrentSlot *prometheus.GaugeVec

	// Lag (slots behind for forward fill, slots to go for backfill)
	Lag *prometheus.GaugeVec
}

func NewConsistencyMetrics(namespace string) *ConsistencyMetrics {
	namespace += "_consistency"

	m := &ConsistencyMetrics{
		CurrentSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_slot",
			Help:      "Current slot being processed",
		}, []string{"process", "relay", "event_type", "network"}),

		Lag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "lag",
			Help:      "Number of slots behind (forward fill) or remaining (backfill)",
		}, []string{"process", "relay", "event_type", "network"}),
	}

	prometheus.MustRegister(m.CurrentSlot, m.Lag)

	return m
}

func (m *ConsistencyMetrics) SetCurrentSlot(process, relay, eventType, network string, slot uint64) {
	m.CurrentSlot.WithLabelValues(process, relay, eventType, network).Set(float64(slot))
}

func (m *ConsistencyMetrics) SetLag(process, relay, eventType, network string, lag int64) {
	if lag < 0 {
		lag = 0
	}

	m.Lag.WithLabelValues(process, relay, eventType, network).Set(float64(lag))
}
