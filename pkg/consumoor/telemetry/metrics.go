package telemetry

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the consumoor service.
type Metrics struct {
	messagesConsumed *prometheus.CounterVec
	messagesRouted   *prometheus.CounterVec
	messagesDropped  *prometheus.CounterVec
	messagesRejected *prometheus.CounterVec
	decodeErrors     *prometheus.CounterVec
	dlqWrites        *prometheus.CounterVec
	dlqErrors        *prometheus.CounterVec
	rowsWritten      *prometheus.CounterVec
	writeErrors      *prometheus.CounterVec
	writeDuration    *prometheus.HistogramVec
	batchSize        *prometheus.HistogramVec
	bufferUsage      *prometheus.GaugeVec
	flattenErrors    *prometheus.CounterVec

	// ch-go pool metrics
	chgoPoolAcquiredResources     prometheus.Gauge
	chgoPoolIdleResources         prometheus.Gauge
	chgoPoolConstructingResources prometheus.Gauge
	chgoPoolTotalResources        prometheus.Gauge
	chgoPoolMaxResources          prometheus.Gauge
	chgoPoolAcquireDuration       prometheus.Gauge
	chgoPoolEmptyAcquireWaitTime  prometheus.Gauge
	chgoPoolAcquireTotal          prometheus.Counter
	chgoPoolEmptyAcquireTotal     prometheus.Counter
	chgoPoolCanceledAcquireTotal  prometheus.Counter
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

		messagesRejected: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "messages_rejected_total",
			Help:      "Total number of permanently rejected messages.",
		}, []string{"reason"}),

		decodeErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "decode_errors_total",
			Help:      "Total number of message decode errors.",
		}, []string{"topic"}),

		dlqWrites: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "dlq_writes_total",
			Help:      "Total number of rejected messages successfully written to DLQ.",
		}, []string{"reason"}),

		dlqErrors: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "dlq_errors_total",
			Help:      "Total number of DLQ publish errors for rejected messages.",
		}, []string{"reason"}),

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

		chgoPoolAcquiredResources: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_acquired_resources",
			Help:      "Number of currently acquired ch-go pool connections.",
		}),

		chgoPoolIdleResources: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_idle_resources",
			Help:      "Number of currently idle ch-go pool connections.",
		}),

		chgoPoolConstructingResources: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_constructing_resources",
			Help:      "Number of currently constructing ch-go pool connections.",
		}),

		chgoPoolTotalResources: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_total_resources",
			Help:      "Total number of ch-go pool connections.",
		}),

		chgoPoolMaxResources: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_max_resources",
			Help:      "Configured maximum number of ch-go pool connections.",
		}),

		chgoPoolAcquireDuration: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_acquire_duration_seconds",
			Help:      "Cumulative time spent acquiring ch-go pool connections.",
		}),

		chgoPoolEmptyAcquireWaitTime: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_empty_acquire_wait_time_seconds",
			Help:      "Cumulative wait time when ch-go pool had no idle connections.",
		}),

		chgoPoolAcquireTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_acquire_total",
			Help:      "Total number of successful ch-go pool acquires.",
		}),

		chgoPoolEmptyAcquireTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_empty_acquire_total",
			Help:      "Total number of acquires that waited for a ch-go pool connection.",
		}),

		chgoPoolCanceledAcquireTotal: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "chgo_pool_canceled_acquire_total",
			Help:      "Total number of canceled ch-go pool acquire attempts.",
		}),
	}
}

func (m *Metrics) MessagesConsumed() *prometheus.CounterVec { return m.messagesConsumed }
func (m *Metrics) MessagesRouted() *prometheus.CounterVec   { return m.messagesRouted }
func (m *Metrics) MessagesDropped() *prometheus.CounterVec  { return m.messagesDropped }
func (m *Metrics) MessagesRejected() *prometheus.CounterVec { return m.messagesRejected }
func (m *Metrics) DecodeErrors() *prometheus.CounterVec     { return m.decodeErrors }
func (m *Metrics) DLQWrites() *prometheus.CounterVec        { return m.dlqWrites }
func (m *Metrics) DLQErrors() *prometheus.CounterVec        { return m.dlqErrors }
func (m *Metrics) RowsWritten() *prometheus.CounterVec      { return m.rowsWritten }
func (m *Metrics) WriteErrors() *prometheus.CounterVec      { return m.writeErrors }
func (m *Metrics) WriteDuration() *prometheus.HistogramVec  { return m.writeDuration }
func (m *Metrics) BatchSize() *prometheus.HistogramVec      { return m.batchSize }
func (m *Metrics) BufferUsage() *prometheus.GaugeVec        { return m.bufferUsage }
func (m *Metrics) FlattenErrors() *prometheus.CounterVec    { return m.flattenErrors }

func (m *Metrics) ChgoPoolAcquiredResources() prometheus.Gauge { return m.chgoPoolAcquiredResources }
func (m *Metrics) ChgoPoolIdleResources() prometheus.Gauge     { return m.chgoPoolIdleResources }
func (m *Metrics) ChgoPoolConstructingResources() prometheus.Gauge {
	return m.chgoPoolConstructingResources
}
func (m *Metrics) ChgoPoolTotalResources() prometheus.Gauge  { return m.chgoPoolTotalResources }
func (m *Metrics) ChgoPoolMaxResources() prometheus.Gauge    { return m.chgoPoolMaxResources }
func (m *Metrics) ChgoPoolAcquireDuration() prometheus.Gauge { return m.chgoPoolAcquireDuration }
func (m *Metrics) ChgoPoolEmptyAcquireWaitTime() prometheus.Gauge {
	return m.chgoPoolEmptyAcquireWaitTime
}
func (m *Metrics) ChgoPoolAcquireTotal() prometheus.Counter      { return m.chgoPoolAcquireTotal }
func (m *Metrics) ChgoPoolEmptyAcquireTotal() prometheus.Counter { return m.chgoPoolEmptyAcquireTotal }
func (m *Metrics) ChgoPoolCanceledAcquireTotal() prometheus.Counter {
	return m.chgoPoolCanceledAcquireTotal
}
