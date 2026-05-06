package execution

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	connectedPeers         *prometheus.GaugeVec
	connectedPeerStartTime *prometheus.GaugeVec
	disconnects            *prometheus.CounterVec
}

var executionMetrics = NewMetrics("xatu_mimicry_execution")

const metricUnknown = "unknown"

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		connectedPeers: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connected_peers",
			Help:      "Current execution P2P peers connected by implementation and network",
		}, []string{"implementation", "network_id"}),
		connectedPeerStartTime: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "connected_peer_start_time_seconds",
			Help:      "Unix timestamp when an execution P2P peer connection reached eth status",
		}, []string{"implementation", "network_id", "node_record"}),
		disconnects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "disconnects_total",
			Help:      "Execution P2P peer disconnects by implementation and reason",
		}, []string{"implementation", "reason"}),
	}

	prometheus.MustRegister(m.connectedPeers)
	prometheus.MustRegister(m.connectedPeerStartTime)
	prometheus.MustRegister(m.disconnects)

	return m
}

func (m *Metrics) IncConnectedPeer(implementation, networkID string) {
	m.connectedPeers.WithLabelValues(normalizeMetricLabel(implementation), normalizeMetricLabel(networkID)).Inc()
}

func (m *Metrics) DecConnectedPeer(implementation, networkID string) {
	m.connectedPeers.WithLabelValues(normalizeMetricLabel(implementation), normalizeMetricLabel(networkID)).Dec()
}

func (m *Metrics) SetConnectedPeerStartTime(implementation, networkID, nodeRecord string, timestamp float64) {
	m.connectedPeerStartTime.WithLabelValues(
		normalizeMetricLabel(implementation),
		normalizeMetricLabel(networkID),
		normalizeMetricLabel(nodeRecord),
	).Set(timestamp)
}

func (m *Metrics) DeleteConnectedPeerStartTime(implementation, networkID, nodeRecord string) {
	m.connectedPeerStartTime.DeleteLabelValues(
		normalizeMetricLabel(implementation),
		normalizeMetricLabel(networkID),
		normalizeMetricLabel(nodeRecord),
	)
}

func (m *Metrics) AddDisconnect(implementation, reason string) {
	m.disconnects.WithLabelValues(normalizeMetricLabel(implementation), normalizeMetricLabel(reason)).Inc()
}

func normalizeMetricLabel(value string) string {
	if value == "" {
		return metricUnknown
	}

	return value
}
