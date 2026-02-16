package consumoor

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the consumoor service.
type Metrics struct {
	messagesConsumed *prometheus.CounterVec
	messagesRouted   *prometheus.CounterVec
	messagesDropped  *prometheus.CounterVec
	decodeErrors     *prometheus.CounterVec
	rowsWritten      *prometheus.CounterVec
	writeErrors      *prometheus.CounterVec
	writeDuration    *prometheus.HistogramVec
	batchSize        *prometheus.HistogramVec
	bufferUsage      *prometheus.GaugeVec
	flattenErrors    *prometheus.CounterVec
}

// NewMetrics creates and registers all consumoor Prometheus metrics.
func NewMetrics(namespace string) *Metrics {
	if namespace == "" {
		namespace = "xatu"
	}

	subsystem := "consumoor"

	return &Metrics{
		messagesConsumed: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_consumed_total",
			Help:      "Total number of Kafka messages consumed.",
		}, []string{"topic"}),

		messagesRouted: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_routed_total",
			Help:      "Total number of messages routed to a flattener.",
		}, []string{"event_name", "table"}),

		messagesDropped: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_dropped_total",
			Help:      "Total number of messages dropped (no flattener, disabled, or filtered).",
		}, []string{"event_name", "reason"}),

		decodeErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "decode_errors_total",
			Help:      "Total number of message decode errors.",
		}, []string{"topic"}),

		rowsWritten: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rows_written_total",
			Help:      "Total number of rows written to ClickHouse.",
		}, []string{"table"}),

		writeErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "write_errors_total",
			Help:      "Total number of ClickHouse write errors.",
		}, []string{"table"}),

		writeDuration: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "write_duration_seconds",
			Help:      "Duration of ClickHouse batch writes.",
			Buckets:   []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30},
		}, []string{"table"}),

		batchSize: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "batch_size",
			Help:      "Number of rows per ClickHouse batch write.",
			Buckets:   []float64{1, 10, 100, 1000, 10000, 50000, 100000, 200000, 500000, 1000000},
		}, []string{"table"}),

		bufferUsage: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "buffer_usage",
			Help:      "Current number of rows buffered per table.",
		}, []string{"table"}),

		flattenErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "flatten_errors_total",
			Help:      "Total number of flattener errors.",
		}, []string{"event_name", "table"}),
	}
}
