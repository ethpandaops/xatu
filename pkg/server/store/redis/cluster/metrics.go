package cluster

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	getTotal    *prometheus.CounterVec
	setTotal    *prometheus.CounterVec
	deleteTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		getTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "get_total",
			Help:      "Total number of get requests",
		}, []string{"type", "status"}),
		setTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "set_total",
			Help:      "Total number of set requests",
		}, []string{"type", "status"}),
		deleteTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "delete_total",
			Help:      "Total number of delete requests",
		}, []string{"type", "status"}),
	}

	prometheus.MustRegister(m.getTotal)
	prometheus.MustRegister(m.setTotal)
	prometheus.MustRegister(m.deleteTotal)

	return m
}

func (m *Metrics) AddGet(count int, cacheType, status string) {
	m.getTotal.WithLabelValues(cacheType, status).Add(float64(count))
}

func (m *Metrics) AddSet(count int, cacheType, status string) {
	m.setTotal.WithLabelValues(cacheType, status).Add(float64(count))
}

func (m *Metrics) AddDelete(count int, cacheType, status string) {
	m.deleteTotal.WithLabelValues(cacheType, status).Add(float64(count))
}
