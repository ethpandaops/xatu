package coordinator

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	nodeRecordsTotal                 *prometheus.CounterVec
	executionNodeRecordStatusesTotal *prometheus.CounterVec
	consensusNodeRecordStatusesTotal *prometheus.CounterVec
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
		}, []string{"sentry_id", "result", "network_id", "fork_id_hash"}),
		consensusNodeRecordStatusesTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "consensus_node_record_statuses_received_total",
			Help:      "Total number of consensus node record statuses received",
		}, []string{"sentry_id", "result", "network_id", "fork_digest"}),
	}

	prometheus.MustRegister(m.nodeRecordsTotal)
	prometheus.MustRegister(m.executionNodeRecordStatusesTotal)
	prometheus.MustRegister(m.consensusNodeRecordStatusesTotal)

	return m
}

func (m *Metrics) AddNodeRecordReceived(count int, sentryID string) {
	m.nodeRecordsTotal.WithLabelValues(sentryID).Add(float64(count))
}

func (m *Metrics) AddExecutionNodeRecordStatusReceived(count int, sentryID, result, networkID, forkIDHash string) {
	m.executionNodeRecordStatusesTotal.WithLabelValues(sentryID, result, networkID, forkIDHash).Add(float64(count))
}

func (m *Metrics) AddConsensusNodeRecordStatusReceived(count int, sentryID, result, networkID, forkDigest string) {
	m.consensusNodeRecordStatusesTotal.WithLabelValues(sentryID, result, networkID, forkDigest).Add(float64(count))
}
