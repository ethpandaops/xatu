package p2p

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	dialedNodeRecordsTotal   *prometheus.CounterVec
	dialFailuresTotal        *prometheus.CounterVec
	activeDialingNodeRecords *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		dialedNodeRecordsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dialed_node_records_total",
			Help:      "Total number of dialed node records",
		}, []string{"result", "layer"}),
		dialFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dial_failures_total",
			Help:      "Total number of failed node record dials by reason",
		}, []string{"layer", "reason"}),
		activeDialingNodeRecords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "active_dialing_node_records",
			Help:      "Total number of active dialing node records",
		}, []string{"layer"}),
	}

	prometheus.MustRegister(m.dialedNodeRecordsTotal)
	prometheus.MustRegister(m.dialFailuresTotal)
	prometheus.MustRegister(m.activeDialingNodeRecords)

	return m
}

func (m *Metrics) AddDialedNodeRecod(count int, result, layer string) {
	m.dialedNodeRecordsTotal.WithLabelValues(result, layer).Add(float64(count))
}

func (m *Metrics) AddDialFailure(count int, layer, reason string) {
	m.dialFailuresTotal.WithLabelValues(layer, reason).Add(float64(count))
}

func (m *Metrics) SetActiveDialingNodeRecods(count int, layer string) {
	m.activeDialingNodeRecords.WithLabelValues(layer).Set(float64(count))
}
