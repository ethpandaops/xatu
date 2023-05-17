package coordinator

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	nodeRecordsTotal                 *prometheus.CounterVec
	executionNodeRecordStatusesTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		nodeRecordsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "node_records_received_total",
			Help:      "Total number of node records received",
		}, []string{"sentry_id"}),
		executionNodeRecordStatusesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "execution_node_record_statuses_received_total",
			Help:      "Total number of execution node record statuses received",
		}, []string{"sentry_id", "result", "fork_id_hash"}),
	}

	prometheus.MustRegister(m.nodeRecordsTotal)
	prometheus.MustRegister(m.executionNodeRecordStatusesTotal)

	return m
}

func (m *Metrics) AddNodeRecordReceived(count int, sentryID string) {
	m.nodeRecordsTotal.WithLabelValues(sentryID).Add(float64(count))
}

func (m *Metrics) AddExecutionNodeRecordStatusReceived(count int, sentryID, result, forkIDHash string) {
	m.executionNodeRecordStatusesTotal.WithLabelValues(sentryID, result, forkIDHash).Add(float64(count))
}
