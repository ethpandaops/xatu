package p2p

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	dialedNodeRecordsTotal   *prometheus.CounterVec
	activeDialingNodeRecords *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		dialedNodeRecordsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dialed_node_records_total",
			Help:      "Total number of dialed node records",
		}, []string{"result"}),
		activeDialingNodeRecords: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "active_dialing_node_records",
			Help:      "Total number of active dialing node records",
		}, []string{"layer"}),
	}

	prometheus.MustRegister(m.dialedNodeRecordsTotal)
	prometheus.MustRegister(m.activeDialingNodeRecords)

	return m
}

func (m *Metrics) AddDialedNodeRecod(count int, result string) {
	m.dialedNodeRecordsTotal.WithLabelValues(result).Add(float64(count))
}

func (m *Metrics) SetActiveDialingNodeRecods(count int, layer string) {
	m.activeDialingNodeRecords.WithLabelValues(layer).Set(float64(count))
}
