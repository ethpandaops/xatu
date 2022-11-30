package coordinator

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	nodeRecordsTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		nodeRecordsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "node_records_received_total",
			Help:      "Total number of node records received",
		}, []string{"sentry_id"}),
	}

	prometheus.MustRegister(m.nodeRecordsTotal)

	return m
}

func (m *Metrics) AddNodeRecordReceived(count int, sentryID string) {
	m.nodeRecordsTotal.WithLabelValues(sentryID).Add(float64(count))
}
