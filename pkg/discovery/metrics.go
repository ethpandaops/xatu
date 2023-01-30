package discovery

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	discoveredNodeRecordsTotal *prometheus.CounterVec
	nodeRecordStatusesTotal    *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		discoveredNodeRecordsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "discovered_node_records_total",
			Help:      "Total number of discovered node records",
		}, []string{"protocol"}),
		nodeRecordStatusesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "node_record_statuses_total",
			Help:      "Total number of node record statuses",
		}, []string{"network", "forkIDHash"}),
	}

	prometheus.MustRegister(m.discoveredNodeRecordsTotal)
	prometheus.MustRegister(m.nodeRecordStatusesTotal)

	return m
}

func (m *Metrics) AddDiscoveredNodeRecord(count int, protocol string) {
	m.discoveredNodeRecordsTotal.WithLabelValues(protocol).Add(float64(count))
}

func (m *Metrics) AddNodeRecordStatus(count int, network, forkIDHash string) {
	m.nodeRecordStatusesTotal.WithLabelValues(network, forkIDHash).Add(float64(count))
}
