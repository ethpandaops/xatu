package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	// DefaultTTL is the default TTL for block deduplication.
	// Set to 13 minutes to cover slightly more than 1 epoch (6.4 minutes)
	// to handle delayed events from multiple beacon nodes.
	DefaultTTL = 13 * time.Minute
)

// DedupCache is a TTL-based cache for deduplicating block events by block root.
// It tracks whether a block root has been seen before to prevent duplicate processing.
type DedupCache struct {
	cache   *ttlcache.Cache[string, time.Time]
	ttl     time.Duration
	metrics *Metrics
}

// Config holds configuration for the deduplication cache.
type Config struct {
	// TTL is the time-to-live for cached entries.
	// After this duration, entries are automatically evicted.
	TTL time.Duration `yaml:"ttl" default:"13m"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.TTL <= 0 {
		c.TTL = DefaultTTL
	}

	return nil
}

// Metrics holds Prometheus metrics for the deduplication cache.
type Metrics struct {
	hitsTotal   prometheus.Counter
	missesTotal prometheus.Counter
	cacheSize   prometheus.Gauge
}

// NewMetrics creates a new Metrics instance for the dedup cache.
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		hitsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dedup_hits_total",
			Help:      "Total number of deduplication cache hits (duplicate blocks dropped)",
		}),
		missesTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "dedup_misses_total",
			Help:      "Total number of deduplication cache misses (new blocks processed)",
		}),
		cacheSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "dedup_cache_size",
			Help:      "Current number of entries in the deduplication cache",
		}),
	}

	prometheus.MustRegister(
		m.hitsTotal,
		m.missesTotal,
		m.cacheSize,
	)

	return m
}

// New creates a new DedupCache with the given configuration and metrics namespace.
func New(cfg *Config, namespace string) *DedupCache {
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = DefaultTTL
	}

	cache := ttlcache.New(
		ttlcache.WithTTL[string, time.Time](ttl),
	)

	return &DedupCache{
		cache:   cache,
		ttl:     ttl,
		metrics: NewMetrics(namespace),
	}
}

// Start begins the cache cleanup goroutine.
// This should be called once when the cache is ready to be used.
func (d *DedupCache) Start() {
	go d.cache.Start()
}

// Stop stops the cache cleanup goroutine.
func (d *DedupCache) Stop() {
	d.cache.Stop()
}

// Check checks if a block root has been seen before.
// Returns true if the block root was already seen (duplicate),
// returns false if the block root is new (first occurrence).
// If the block root is new, it is automatically added to the cache.
func (d *DedupCache) Check(blockRoot string) bool {
	// Try to get the existing entry
	item := d.cache.Get(blockRoot)
	if item != nil {
		// Block root was already seen - this is a duplicate
		d.metrics.hitsTotal.Inc()

		return true
	}

	// Block root is new - add it to the cache
	d.cache.Set(blockRoot, time.Now(), d.ttl)
	d.metrics.missesTotal.Inc()
	d.metrics.cacheSize.Set(float64(d.cache.Len()))

	return false
}

// Size returns the current number of entries in the cache.
func (d *DedupCache) Size() int {
	return d.cache.Len()
}

// TTL returns the configured TTL for cache entries.
func (d *DedupCache) TTL() time.Duration {
	return d.ttl
}
