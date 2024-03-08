package ethereum

import "github.com/prometheus/client_golang/prometheus"

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
	// The number of blob sidecars that have been fetched.
	blobSidecarsFetched *prometheus.CounterVec
	// The number of blob sidecars fetches that have failed.
	blobSidecarsFetchErrors *prometheus.CounterVec
	// blobSidecarsCacheHit is the number of times a blob sidecars was found in the cache.
	blobSidecarsCacheHit *prometheus.CounterVec
	// blobSidecarsCacheMiss is the number of times a blob sidecars was not found in the cache.
	blobSidecarsCacheMiss *prometheus.CounterVec
	// preloadBlobSidecarsQueueSize is the number of blob sidecars in the preload queue.
	preloadBlobSidecarsQueueSize *prometheus.GaugeVec
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
		blobSidecarsFetched: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blob_sidecars_fetched_total",
			Help:      "The number of blob sidecars that have been fetched",
		}, []string{"network", "beacon"}),
		blobSidecarsFetchErrors: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blob_sidecars_fetch_errors_total",
			Help:      "The number of blob sidecars that have failed to be fetched",
		}, []string{"network", "beacon"}),
		blobSidecarsCacheHit: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blob_sidecars_cache_hit_total",
			Help:      "The number of times a blob sidecars from a block was found in the cache",
		}, []string{"network", "beacon"}),
		blobSidecarsCacheMiss: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blob_sidecars_cache_miss_total",
			Help:      "The number of times a blob sidecars from a block was not found in the cache",
		}, []string{"network", "beacon"}),
		preloadBlobSidecarsQueueSize: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "preload_blob_sidecars_queue_size",
			Help:      "The number of blob sidecars in the preload queue",
		}, []string{"network", "beacon"}),
	}

	prometheus.MustRegister(m.blocksFetched)
	prometheus.MustRegister(m.blocksFetchErrors)
	prometheus.MustRegister(m.blockCacheHit)
	prometheus.MustRegister(m.blockCacheMiss)
	prometheus.MustRegister(m.preloadBlockQueueSize)
	prometheus.MustRegister(m.blobSidecarsFetched)
	prometheus.MustRegister(m.blobSidecarsFetchErrors)
	prometheus.MustRegister(m.blobSidecarsCacheHit)
	prometheus.MustRegister(m.blobSidecarsCacheMiss)
	prometheus.MustRegister(m.preloadBlobSidecarsQueueSize)

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

func (m *Metrics) IncBlobSidecarsFetched(network string) {
	m.blobSidecarsFetched.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) IncBlobSidecarsFetchErrors(network string) {
	m.blobSidecarsFetchErrors.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) IncBlobSidecarsCacheHit(network string) {
	m.blobSidecarsCacheHit.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) IncBlobSidecarsCacheMiss(network string) {
	m.blobSidecarsCacheMiss.WithLabelValues(network, m.beacon).Inc()
}

func (m *Metrics) SetPreloadBlobSidecarsQueueSize(network string, size int) {
	m.preloadBlobSidecarsQueueSize.WithLabelValues(network, m.beacon).Set(float64(size))
}
