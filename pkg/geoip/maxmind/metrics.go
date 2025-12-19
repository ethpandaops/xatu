package maxmind

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	lookupIPTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		lookupIPTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "lookup_ip_total",
			Help:      "Total number of lookup by IP requests",
		}, []string{"type", "status"}),
	}

	prometheus.MustRegister(m.lookupIPTotal)

	return m
}

func (m *Metrics) AddLookupIP(count int, providerType, status string) {
	m.lookupIPTotal.WithLabelValues(providerType, status).Add(float64(count))
}
