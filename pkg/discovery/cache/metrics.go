package cache

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	duplicateInsertionsTotal *prometheus.GaugeVec
	duplicateHitsTotal       *prometheus.GaugeVec
	duplicateMissesTotal     *prometheus.GaugeVec
	duplicateEvictionsTotal  *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		duplicateInsertionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "duplicate_insertions_total",
			Help:      "Total number of duplicate insertions",
		}, []string{"store"}),
		duplicateHitsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "duplicate_hits_total",
			Help:      "Total number of duplicate hits",
		}, []string{"store"}),
		duplicateMissesTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "duplicate_misses_total",
			Help:      "Total number of duplicate misses",
		}, []string{"store"}),
		duplicateEvictionsTotal: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "duplicate_evictions_total",
			Help:      "Total number of duplicate evictions",
		}, []string{"store"}),
	}

	prometheus.MustRegister(m.duplicateInsertionsTotal)
	prometheus.MustRegister(m.duplicateHitsTotal)
	prometheus.MustRegister(m.duplicateMissesTotal)
	prometheus.MustRegister(m.duplicateEvictionsTotal)

	return m
}

func (m *Metrics) SetDuplicateInsertions(count uint64, store string) {
	m.duplicateInsertionsTotal.WithLabelValues(store).Set(float64(count))
}

func (m *Metrics) SetDuplicateHits(count uint64, store string) {
	m.duplicateHitsTotal.WithLabelValues(store).Set(float64(count))
}

func (m *Metrics) SetDuplicateMisses(count uint64, store string) {
	m.duplicateMissesTotal.WithLabelValues(store).Set(float64(count))
}

func (m *Metrics) SetDuplicateEvictions(count uint64, store string) {
	m.duplicateEvictionsTotal.WithLabelValues(store).Set(float64(count))
}
