package processor

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DefaultMetrics = NewMetrics("xatu")
)

type Metrics struct {
	itemsQueued        *prometheus.GaugeVec
	itemsDropped       *prometheus.CounterVec
	itemsFailed        *prometheus.CounterVec
	itemsExported      *prometheus.CounterVec
	exportDuration     *prometheus.HistogramVec
	itemsProcessed     *prometheus.CounterVec
	batchesProcessed   *prometheus.CounterVec
	processingDuration *prometheus.HistogramVec
	queueWaitTime      *prometheus.HistogramVec
	batchSize          *prometheus.HistogramVec
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
		itemsFailed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "items_failed_total",
			Namespace: namespace,
			Help:      "Number of items failed",
		}, []string{"processor"}),
		itemsExported: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "items_exported_total",
			Namespace: namespace,
			Help:      "Number of items exported",
		}, []string{"processor"}),
		exportDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:      "export_duration_seconds",
			Namespace: namespace,
			Help:      "Duration of export operations in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
		}, []string{"processor"}),

		itemsProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "items_processed_total",
			Namespace: namespace,
			Help:      "Number of items processed",
		}, []string{"processor"}),
		batchesProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "batches_processed_total",
			Namespace: namespace,
			Help:      "Number of batches processed",
		}, []string{"processor"}),
		queueWaitTime: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:      "queue_wait_time_seconds",
			Namespace: namespace,
			Help:      "Time items spend waiting in the queue in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
		}, []string{"processor"}),
		batchSize: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:      "batch_size",
			Namespace: namespace,
			Help:      "Size of processed batches",
			Buckets:   prometheus.ExponentialBuckets(0.1, 2, 10),
		}, []string{"processor"}),
	}

	prometheus.MustRegister(m.itemsQueued)
	prometheus.MustRegister(m.itemsDropped)
	prometheus.MustRegister(m.itemsFailed)
	prometheus.MustRegister(m.itemsExported)
	prometheus.MustRegister(m.exportDuration)
	prometheus.MustRegister(m.itemsProcessed)
	prometheus.MustRegister(m.batchesProcessed)
	prometheus.MustRegister(m.queueWaitTime)
	prometheus.MustRegister(m.batchSize)

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

func (m *Metrics) IncItemsFailedBy(name string, count float64) {
	m.itemsFailed.WithLabelValues(name).Add(count)
}

func (m *Metrics) ObserveExportDuration(name string, duration float64) {
	m.exportDuration.WithLabelValues(name).Observe(duration)
}

func (m *Metrics) IncItemsProcessedBy(name string, count float64) {
	m.itemsProcessed.WithLabelValues(name).Add(count)
}

func (m *Metrics) IncBatchesProcessedBy(name string, count float64) {
	m.batchesProcessed.WithLabelValues(name).Add(count)
}

func (m *Metrics) ObserveQueueWaitTime(name string, duration float64) {
	m.queueWaitTime.WithLabelValues(name).Observe(duration)
}

func (m *Metrics) ObserveBatchSize(name string, size float64) {
	m.batchSize.WithLabelValues(name).Observe(size)
}
