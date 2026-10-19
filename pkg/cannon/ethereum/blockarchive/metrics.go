package blockarchive

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const labelNetwork = "network"

// Metrics are the archive block source's Prometheus collectors.
type Metrics struct {
	lookups       *prometheus.CounterVec
	fetches       *prometheus.CounterVec
	fetchDuration *prometheus.HistogramVec
	fetchedBytes  *prometheus.CounterVec
	blocks        *prometheus.CounterVec
	envelopes     *prometheus.CounterVec
}

// NewMetrics creates and registers the archive metrics under namespace.
func NewMetrics(namespace string) *Metrics {
	namespace += "_block_archive"

	m := &Metrics{
		lookups: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "lookups_total",
			Help:      "Era archive lookups for a slot by result (hit, not_covered, not_canonical, unavailable, error)",
		}, []string{labelNetwork, "result"}),
		fetches: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "fetches_total",
			Help:      "Era fetches by mode (era, range) and result (ok, error)",
		}, []string{labelNetwork, "mode", "result"}),
		fetchDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "fetch_duration_seconds",
			Help:      "Time to download and decode a whole era or an era range",
			Buckets:   []float64{0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30, 60, 120},
		}, []string{labelNetwork, "mode"}),
		fetchedBytes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "fetched_bytes_total",
			Help:      "Bytes downloaded from the archive",
		}, []string{labelNetwork}),
		blocks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blocks_decoded_total",
			Help:      "Beacon blocks decoded from era files",
		}, []string{labelNetwork}),
		envelopes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "envelopes_decoded_total",
			Help:      "Execution payload envelopes decoded from era envelope sidecars",
		}, []string{labelNetwork}),
	}

	prometheus.MustRegister(m.lookups, m.fetches, m.fetchDuration, m.fetchedBytes, m.blocks, m.envelopes)

	return m
}

func (m *Metrics) lookup(network, result string) {
	m.lookups.WithLabelValues(network, result).Inc()
}

func (m *Metrics) fetched(network, mode string, d time.Duration, size int64, slots Slots, err error) {
	result := "ok"
	if err != nil {
		result = "error"
	}

	var blocks, envelopes int

	for _, s := range slots {
		if s.Block != nil {
			blocks++
		}

		if s.Envelope != nil {
			envelopes++
		}
	}

	m.fetches.WithLabelValues(network, mode, result).Inc()
	m.fetchDuration.WithLabelValues(network, mode).Observe(d.Seconds())
	m.fetchedBytes.WithLabelValues(network).Add(float64(size))
	m.blocks.WithLabelValues(network).Add(float64(blocks))
	m.envelopes.WithLabelValues(network).Add(float64(envelopes))
}
