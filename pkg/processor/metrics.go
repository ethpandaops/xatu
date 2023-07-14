package processor

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	name string

	itemsQueued   *prometheus.GaugeVec
	itemsDropped  *prometheus.GaugeVec
	itemsExported *prometheus.CounterVec
}

func NewMetrics(namespace, name string) (*Metrics, error) {
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
		itemsDropped: prometheus.NewGaugeVec(prometheus.GaugeOpts{
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

	dupMetricsCollectorError := "duplicate metrics collector registration attempted"

	if err := prometheus.Register(m.itemsQueued); err != nil && err.Error() != dupMetricsCollectorError {
		return nil, err
	}

	if err := prometheus.Register(m.itemsDropped); err != nil && err.Error() != dupMetricsCollectorError {
		return nil, err
	}

	if err := prometheus.Register(m.itemsExported); err != nil && err.Error() != dupMetricsCollectorError {
		return nil, err
	}

	m.name = name

	return m, nil
}

func (m *Metrics) SetItemsQueued(count float64) {
	m.itemsQueued.WithLabelValues(m.name).Set(count)
}

func (m *Metrics) SetItemsDropped(count float64) {
	m.itemsDropped.WithLabelValues(m.name).Set(count)
}

func (m *Metrics) IncItemsExportedBy(count float64) {
	m.itemsExported.WithLabelValues(m.name).Add(count)
}
