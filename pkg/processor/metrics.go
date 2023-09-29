package processor

import "github.com/prometheus/client_golang/prometheus"

var (
	DefaultMetrics = NewMetrics("xatu")
)

type Metrics struct {
	itemsQueued   *prometheus.GaugeVec
	itemsDropped  *prometheus.CounterVec
	itemsExported *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	if namespace != "" {
		namespace += "_"
	}

	namespace += "processor"

	m := &Metrics{
		itemsQueued: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name:      "items_queued",
			Namespace: namespace,
			Help:      "Number of items queued",
		}, []string{"processor"}),
		itemsDropped: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "items_dropped_total",
			Namespace: namespace,
			Help:      "Number of items dropped",
		}, []string{"processor"}),
		itemsExported: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "items_exported_total",
			Namespace: namespace,
			Help:      "Number of items exported",
		}, []string{"processor"}),
	}

	prometheus.MustRegister(m.itemsQueued)
	prometheus.MustRegister(m.itemsDropped)
	prometheus.MustRegister(m.itemsExported)

	return m
}

func (m *Metrics) SetItemsQueued(name string, count float64) {
	m.itemsQueued.WithLabelValues(name).Set(count)
}

func (m *Metrics) IncItemsDroppedBy(name string, count float64) {
	m.itemsDropped.WithLabelValues(name).Add(count)
}

func (m *Metrics) IncItemsExportedBy(name string, count float64) {
	m.itemsExported.WithLabelValues(name).Add(count)
}
