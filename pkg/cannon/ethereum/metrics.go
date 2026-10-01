package ethereum

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	beacon string
	// The number of blocks that have been fetched.
	blocksFetched *prometheus.CounterVec
	// The number of blocks fetches that have failed.
	blocksFetchErrors *prometheus.CounterVec
	// BlockCacheHit is the number of times a block was found in the cache.
	blockCacheHit *prometheus.CounterVec
	// BlockCacheMiss is the number of times a block was not found in the cache.
	blockCacheMiss *prometheus.CounterVec
	// PreloadBlockQueueSize is the number of blocks in the preload queue.
	preloadBlockQueueSize *prometheus.GaugeVec
	// BeaconStateCacheHit is the number of times a beacon state was served from the cache.
	beaconStateCacheHit *prometheus.CounterVec
	// BeaconStateFetchDuration is the time taken to download and decode a beacon state.
	beaconStateFetchDuration *prometheus.HistogramVec
}

func NewMetrics(namespace, beaconNodeName string) *Metrics {
	namespace += "_ethereum"

	m := &Metrics{
		beacon: beaconNodeName,
		blocksFetched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blocks_fetched_total",
			Help:      "The number of blocks that have been fetched",
		}, []string{"network", "beacon"}),
		blocksFetchErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blocks_fetch_errors_total",
			Help:      "The number of blocks that have failed to be fetched",
		}, []string{"network", "beacon"}),
		blockCacheHit: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "block_cache_hit_total",
			Help:      "The number of times a block was found in the cache",
		}, []string{"network", "beacon"}),
		blockCacheMiss: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "block_cache_miss_total",
			Help:      "The number of times a block was not found in the cache",
		}, []string{"network", "beacon"}),
		preloadBlockQueueSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "preload_block_queue_size",
			Help:      "The number of blocks in the preload queue",
		}, []string{"network", "beacon"}),
		beaconStateCacheHit: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "beacon_state_cache_hit_total",
			Help:      "The number of times a beacon state was served from the cache",
		}, []string{"network", "beacon"}),
		beaconStateFetchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "beacon_state_fetch_duration_seconds",
			Help:      "The time taken to download and decode a beacon state",
			Buckets:   []float64{1, 2, 5, 10, 20, 30, 60, 120, 300},
		}, []string{"network", "beacon"}),
	}

	prometheus.MustRegister(m.blocksFetched)
	prometheus.MustRegister(m.blocksFetchErrors)
	prometheus.MustRegister(m.blockCacheHit)
	prometheus.MustRegister(m.blockCacheMiss)
	prometheus.MustRegister(m.preloadBlockQueueSize)
	prometheus.MustRegister(m.beaconStateCacheHit)
	prometheus.MustRegister(m.beaconStateFetchDuration)

	return m
}

func (m *Metrics) IncBlocksFetched(network string) {
	m.blocksFetched.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) IncBlocksFetchErrors(network string) {
	m.blocksFetchErrors.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) IncBlockCacheHit(network string) {
	m.blockCacheHit.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) IncBlockCacheMiss(network string) {
	m.blockCacheMiss.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) SetPreloadBlockQueueSize(network string, size int) {
	m.preloadBlockQueueSize.WithLabelValues(network, m.beacon).Set(float64(size))
}

func (m *Metrics) IncBeaconStateCacheHit(network string) {
	m.beaconStateCacheHit.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) ObserveBeaconStateFetch(network string, d time.Duration) {
	m.beaconStateFetchDuration.WithLabelValues(network, m.beacon).Observe(d.Seconds())
}
