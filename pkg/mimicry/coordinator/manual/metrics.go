package manual

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	peersTotal *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		peersTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "peers_total",
			Help:      "Total number of peers",
		}, []string{"status"}),
	}

	prometheus.MustRegister(m.peersTotal)

	return m
}

func (m *Metrics) SetPeers(count int, status string) {
	m.peersTotal.WithLabelValues(status).Set(float64(count))
}
