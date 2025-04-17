package execution

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	networkID             string
	txWatcherReceived     *prometheus.CounterVec
	txWatcherProcessed    *prometheus.CounterVec
	txWatcherExpired      *prometheus.CounterVec
	txWatcherPendingGauge *prometheus.GaugeVec

	// Queue metrics
	queueSizeGauge         *prometheus.GaugeVec
	queueRejectionsCounter *prometheus.CounterVec
	queueThroughputCounter *prometheus.CounterVec

	// Processing time metrics
	rpcRequestDurationHistogram      *prometheus.HistogramVec
	batchProcessingDurationHistogram *prometheus.HistogramVec

	// Additional metrics
	txBySourceCounter          *prometheus.CounterVec
	txProcessingOutcomeCounter *prometheus.CounterVec
	websocketConnectionGauge   *prometheus.GaugeVec
	circuitBreakerStateGauge   *prometheus.GaugeVec
	processedCacheSizeGauge    *prometheus.GaugeVec
}

func NewMetrics(namespace, networkID string) *Metrics {
	m := &Metrics{
		networkID: networkID,
		// Initialize transaction watcher metrics
		txWatcherReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_watcher_received_total",
			Help:      "Total number of transactions received by the watcher",
		}, []string{"network_id"}),

		txWatcherProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_watcher_processed_total",
			Help:      "Total number of transactions processed by the watcher",
		}, []string{"network_id"}),

		txWatcherExpired: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_watcher_expired_total",
			Help:      "Total number of transactions expired in the watcher",
		}, []string{"network_id"}),

		txWatcherPendingGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tx_watcher_pending",
			Help:      "Current number of pending transactions in the watcher",
		}, []string{"network_id"}),

		// Queue metrics
		queueSizeGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tx_queue_size",
			Help:      "Current number of transactions in the processing queue",
		}, []string{"network_id"}),

		queueRejectionsCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_queue_rejections_total",
			Help:      "Total number of transactions rejected due to queue being full",
		}, []string{"network_id"}),

		// New metrics

		queueThroughputCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_queue_throughput_total",
			Help:      "Total number of transactions that have passed through the queue",
		}, []string{"network_id"}),

		// Processing time metrics
		rpcRequestDurationHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "tx_rpc_request_duration_seconds",
			Help:      "Duration of RPC requests in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.01, 2, 15), // From 10ms to ~160s (extended from 10 to 15 buckets)
		}, []string{"network_id", "method"}),

		batchProcessingDurationHistogram: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "tx_batch_processing_duration_seconds",
			Help:      "Duration of batch processing in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.001, 2, 10), // From 1ms to ~1s
		}, []string{"network_id"}),

		// Additional metrics
		txBySourceCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_by_source_total",
			Help:      "Total number of transactions by source",
		}, []string{"network_id", "source"}),

		txProcessingOutcomeCounter: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "tx_processing_outcome_total",
			Help:      "Outcome of transaction processing",
		}, []string{"network_id", "outcome"}),

		websocketConnectionGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tx_websocket_connected",
			Help:      "Whether websocket is currently connected (1) or disconnected (0)",
		}, []string{"network_id"}),

		circuitBreakerStateGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tx_circuit_breaker_state",
			Help:      "Current state of the circuit breaker (0=closed, 1=half-open, 2=open)",
		}, []string{"network_id", "breaker_name"}),

		processedCacheSizeGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "tx_processed_cache_size",
			Help:      "Current size of the processed transaction cache",
		}, []string{"network_id"}),
	}

	prometheus.MustRegister(
		m.txWatcherReceived,
		m.txWatcherProcessed,
		m.txWatcherExpired,
		m.txWatcherPendingGauge,
		// Register new metrics
		m.queueSizeGauge,
		m.queueRejectionsCounter,
		m.queueThroughputCounter,
		m.rpcRequestDurationHistogram,
		m.batchProcessingDurationHistogram,
		m.txBySourceCounter,
		m.txProcessingOutcomeCounter,
		m.websocketConnectionGauge,
		m.circuitBreakerStateGauge,
		m.processedCacheSizeGauge,
	)

	return m
}

// AddMempoolTxReceived increments the counter for received mempool transactions
func (m *Metrics) AddMempoolTxReceived(count int) {
	m.txWatcherReceived.WithLabelValues(m.networkID).Add(float64(count))
}

// AddMempoolTxProcessed increments the counter for processed mempool transactions
func (m *Metrics) AddMempoolTxProcessed(count int) {
	m.txWatcherProcessed.WithLabelValues(m.networkID).Add(float64(count))
}

// AddMempoolTxExpired increments the counter for expired mempool transactions
func (m *Metrics) AddMempoolTxExpired(count int) {
	m.txWatcherExpired.WithLabelValues(m.networkID).Add(float64(count))
}

// SetMempoolTxPending sets the gauge for pending mempool transactions
func (m *Metrics) SetMempoolTxPending(count int) {
	m.txWatcherPendingGauge.WithLabelValues(m.networkID).Set(float64(count))
}

// SetQueueSize sets the queue size gauge
func (m *Metrics) SetQueueSize(size int) {
	if m == nil {
		return
	}

	m.queueSizeGauge.WithLabelValues(m.networkID).Set(float64(size))
}

func (m *Metrics) AddQueueRejections(count int) {
	m.queueRejectionsCounter.WithLabelValues(m.networkID).Add(float64(count))
}

// New method for queue throughput
func (m *Metrics) AddQueueThroughput(count int) {
	m.queueThroughputCounter.WithLabelValues(m.networkID).Add(float64(count))
}

// New methods for processing time metrics
func (m *Metrics) ObserveRPCRequestDuration(method string, duration float64) {
	m.rpcRequestDurationHistogram.WithLabelValues(m.networkID, method).Observe(duration)
}

func (m *Metrics) ObserveBatchProcessingDuration(duration float64) {
	m.batchProcessingDurationHistogram.WithLabelValues(m.networkID).Observe(duration)
}

// New methods for additional metrics
func (m *Metrics) AddTxBySource(source string, count int) {
	m.txBySourceCounter.WithLabelValues(m.networkID, source).Add(float64(count))
}

func (m *Metrics) AddTxProcessingOutcome(outcome string, count int) {
	m.txProcessingOutcomeCounter.WithLabelValues(m.networkID, outcome).Add(float64(count))
}

func (m *Metrics) SetWebsocketConnected(connected bool) {
	value := 0.0
	if connected {
		value = 1.0
	}
	m.websocketConnectionGauge.WithLabelValues(m.networkID).Set(value)
}

func (m *Metrics) SetProcessedCacheSize(size int) {
	m.processedCacheSizeGauge.WithLabelValues(m.networkID).Set(float64(size))
}

// SetCircuitBreakerState updates the circuit breaker state gauge
func (m *Metrics) SetCircuitBreakerState(breakerName, state string) {
	// Convert state string to numeric value
	var stateValue float64
	switch state {
	case "closed":
		stateValue = 0
	case "half-open":
		stateValue = 1
	case "open":
		stateValue = 2
	default:
		stateValue = -1 // Unknown state
	}

	m.circuitBreakerStateGauge.WithLabelValues(m.networkID, breakerName).Set(stateValue)
}
