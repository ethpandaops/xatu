package sentry

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec

	// Mempool transaction metrics
	mempoolTxReceived     *prometheus.CounterVec
	mempoolTxProcessed    *prometheus.CounterVec
	mempoolTxExpired      *prometheus.CounterVec
	mempoolTxNull         *prometheus.CounterVec
	mempoolTxPendingGauge *prometheus.GaugeVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events received",
		}, []string{"type", "network_id"}),

		// Initialize mempool transaction metrics
		mempoolTxReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "mempool_tx_received_total",
			Help:      "Total number of mempool transactions received",
		}, []string{"network_id"}),

		mempoolTxProcessed: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "mempool_tx_processed_total",
			Help:      "Total number of mempool transactions processed",
		}, []string{"network_id"}),

		mempoolTxExpired: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "mempool_tx_expired_total",
			Help:      "Total number of mempool transactions expired",
		}, []string{"network_id"}),

		mempoolTxNull: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "mempool_tx_null_total",
			Help:      "Total number of mempool transactions that returned null",
		}, []string{"network_id"}),

		mempoolTxPendingGauge: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "mempool_tx_pending",
			Help:      "Current number of pending mempool transactions",
		}, []string{"network_id"}),
	}

	prometheus.MustRegister(
		m.decoratedEventTotal,
		m.mempoolTxReceived,
		m.mempoolTxProcessed,
		m.mempoolTxExpired,
		m.mempoolTxNull,
		m.mempoolTxPendingGauge,
	)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType, networkID string) {
	m.decoratedEventTotal.WithLabelValues(eventType, networkID).Add(float64(count))
}

// AddMempoolTxReceived increments the counter for received mempool transactions
func (m *Metrics) AddMempoolTxReceived(count int, networkID string) {
	m.mempoolTxReceived.WithLabelValues(networkID).Add(float64(count))
}

// AddMempoolTxProcessed increments the counter for processed mempool transactions
func (m *Metrics) AddMempoolTxProcessed(count int, networkID string) {
	m.mempoolTxProcessed.WithLabelValues(networkID).Add(float64(count))
}

// AddMempoolTxExpired increments the counter for expired mempool transactions
func (m *Metrics) AddMempoolTxExpired(count int, networkID string) {
	m.mempoolTxExpired.WithLabelValues(networkID).Add(float64(count))
}

// AddMempoolTxNull increments the counter for null mempool transactions
func (m *Metrics) AddMempoolTxNull(count int, networkID string) {
	m.mempoolTxNull.WithLabelValues(networkID).Add(float64(count))
}

// SetMempoolTxPending sets the gauge for pending mempool transactions
func (m *Metrics) SetMempoolTxPending(count int, networkID string) {
	m.mempoolTxPendingGauge.WithLabelValues(networkID).Set(float64(count))
}
