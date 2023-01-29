package xatu

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	peersTotal                  *prometheus.GaugeVec
	peerConnectionAttemptsTotal *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		peersTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "peers_total",
			Help:      "Total number of peers",
		}, []string{"status"}),
		peerConnectionAttemptsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "peer_connection_attempts_total",
			Help:      "Total number of peer connection attempts",
		}, []string{}),
	}

	prometheus.MustRegister(m.peersTotal)
	prometheus.MustRegister(m.peerConnectionAttemptsTotal)

	return m
}

func (m *Metrics) SetPeers(count int, status string) {
	m.peersTotal.WithLabelValues(status).Set(float64(count))
}

func (m *Metrics) SetPeerConnectionAttempts(count int) {
	m.peerConnectionAttemptsTotal.WithLabelValues().Set(float64(count))
}
