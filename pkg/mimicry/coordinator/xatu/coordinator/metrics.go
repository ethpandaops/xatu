package coordinator

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	nodeRecordStatusesTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		nodeRecordStatusesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "node_record_statuses_total",
			Help:      "Total number of node record statuses",
		}, []string{"network", "forkIDHash"}),
	}

	prometheus.MustRegister(m.nodeRecordStatusesTotal)

	return m
}

func (m *Metrics) AddNodeRecordStatus(count int, network, forkIDHash string) {
	m.nodeRecordStatusesTotal.WithLabelValues(network, forkIDHash).Add(float64(count))
}
