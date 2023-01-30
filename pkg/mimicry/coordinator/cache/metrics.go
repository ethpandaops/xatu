package cache

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	sharedInsertionsTotal *prometheus.GaugeVec
	sharedHitsTotal       *prometheus.GaugeVec
	sharedMissesTotal     *prometheus.GaugeVec
	sharedEvictionsTotal  *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		sharedInsertionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "shared_insertions_total",
			Help:      "Total number of shared insertions",
		}, []string{"store"}),
		sharedHitsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "shared_hits_total",
			Help:      "Total number of shared hits",
		}, []string{"store"}),
		sharedMissesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "shared_misses_total",
			Help:      "Total number of shared misses",
		}, []string{"store"}),
		sharedEvictionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "shared_evictions_total",
			Help:      "Total number of shared evictions",
		}, []string{"store"}),
	}

	prometheus.MustRegister(m.sharedInsertionsTotal)
	prometheus.MustRegister(m.sharedHitsTotal)
	prometheus.MustRegister(m.sharedMissesTotal)
	prometheus.MustRegister(m.sharedEvictionsTotal)

	return m
}

func (m *Metrics) SetSharedInsertions(count uint64, store string) {
	m.sharedInsertionsTotal.WithLabelValues(store).Set(float64(count))
}

func (m *Metrics) SetSharedHits(count uint64, store string) {
	m.sharedHitsTotal.WithLabelValues(store).Set(float64(count))
}

func (m *Metrics) SetSharedMisses(count uint64, store string) {
	m.sharedMissesTotal.WithLabelValues(store).Set(float64(count))
}

func (m *Metrics) SetSharedEvictions(count uint64, store string) {
	m.sharedEvictionsTotal.WithLabelValues(store).Set(float64(count))
}
