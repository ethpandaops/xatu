package ethereum

import (
	"github.com/prometheus/client_golang/prometheus"
)

// BeaconNodeStatus represents the health status of a beacon node.
type BeaconNodeStatus string

const (
	// BeaconNodeStatusHealthy indicates the beacon node is healthy.
	BeaconNodeStatusHealthy BeaconNodeStatus = "healthy"
	// BeaconNodeStatusUnhealthy indicates the beacon node is unhealthy.
	BeaconNodeStatusUnhealthy BeaconNodeStatus = "unhealthy"
	// BeaconNodeStatusConnecting indicates the beacon node is connecting.
	BeaconNodeStatusConnecting BeaconNodeStatus = "connecting"
)

// Metrics holds Prometheus metrics for the beacon node pool.
type Metrics struct {
	// beaconNodeStatus tracks the status of each beacon node (1 = status active, 0 = not).
	beaconNodeStatus *prometheus.GaugeVec

	// blocksFetched tracks the total number of blocks fetched per beacon node.
	blocksFetched *prometheus.CounterVec

	// blockCacheHits tracks the number of block cache hits.
	blockCacheHits *prometheus.CounterVec

	// blockCacheMisses tracks the number of block cache misses.
	blockCacheMisses *prometheus.CounterVec

	// blockFetchErrors tracks the number of block fetch errors.
	blockFetchErrors *prometheus.CounterVec

	// healthCheckTotal tracks the total number of health checks per node.
	healthCheckTotal *prometheus.CounterVec

	// healthCheckDuration tracks the duration of health checks.
	healthCheckDuration *prometheus.HistogramVec
}

// NewMetrics creates a new Metrics instance.
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		beaconNodeStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "beacon_node_status",
			Help:      "Status of the beacon node (1 = status is active for this node)",
		}, []string{"node", "status"}),

		blocksFetched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "beacon_blocks_fetched_total",
			Help:      "Total number of blocks fetched from beacon nodes",
		}, []string{"node", "network"}),

		blockCacheHits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "beacon_block_cache_hits_total",
			Help:      "Total number of block cache hits",
		}, []string{"network"}),

		blockCacheMisses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "beacon_block_cache_misses_total",
			Help:      "Total number of block cache misses",
		}, []string{"network"}),

		blockFetchErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "beacon_block_fetch_errors_total",
			Help:      "Total number of block fetch errors",
		}, []string{"node", "network"}),

		healthCheckTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "beacon_health_check_total",
			Help:      "Total number of health checks per beacon node",
		}, []string{"node", "status"}),

		healthCheckDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "beacon_health_check_duration_seconds",
			Help:      "Duration of health checks in seconds",
			Buckets:   prometheus.DefBuckets,
		}, []string{"node"}),
	}

	prometheus.MustRegister(
		m.beaconNodeStatus,
		m.blocksFetched,
		m.blockCacheHits,
		m.blockCacheMisses,
		m.blockFetchErrors,
		m.healthCheckTotal,
		m.healthCheckDuration,
	)

	return m
}

// SetBeaconNodeStatus sets the status of a beacon node.
// It sets the gauge to 1 for the current status and 0 for other statuses.
func (m *Metrics) SetBeaconNodeStatus(node string, status BeaconNodeStatus) {
	statuses := []BeaconNodeStatus{
		BeaconNodeStatusHealthy,
		BeaconNodeStatusUnhealthy,
		BeaconNodeStatusConnecting,
	}

	for _, s := range statuses {
		if s == status {
			m.beaconNodeStatus.WithLabelValues(node, string(s)).Set(1)
		} else {
			m.beaconNodeStatus.WithLabelValues(node, string(s)).Set(0)
		}
	}
}

// IncBlocksFetched increments the blocks fetched counter.
func (m *Metrics) IncBlocksFetched(node, network string) {
	m.blocksFetched.WithLabelValues(node, network).Inc()
}

// IncBlockCacheHits increments the block cache hits counter.
func (m *Metrics) IncBlockCacheHits(network string) {
	m.blockCacheHits.WithLabelValues(network).Inc()
}

// IncBlockCacheMisses increments the block cache misses counter.
func (m *Metrics) IncBlockCacheMisses(network string) {
	m.blockCacheMisses.WithLabelValues(network).Inc()
}

// IncBlockFetchErrors increments the block fetch errors counter.
func (m *Metrics) IncBlockFetchErrors(node, network string) {
	m.blockFetchErrors.WithLabelValues(node, network).Inc()
}

// IncHealthCheck increments the health check counter.
func (m *Metrics) IncHealthCheck(node string, status BeaconNodeStatus) {
	m.healthCheckTotal.WithLabelValues(node, string(status)).Inc()
}

// ObserveHealthCheckDuration observes the duration of a health check.
func (m *Metrics) ObserveHealthCheckDuration(node string, duration float64) {
	m.healthCheckDuration.WithLabelValues(node).Observe(duration)
}
