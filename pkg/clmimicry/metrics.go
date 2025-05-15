package clmimicry

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type Metrics struct {
	// Events metrics.
	decoratedEvents *prometheus.CounterVec

	// Sampling metrics.
	samplingProcessed *prometheus.CounterVec
	samplingSkipped   *prometheus.CounterVec

	// Sharding metrics.
	shardDistribution     *prometheus.CounterVec
	shardProcessed        *prometheus.CounterVec
	shardSkipped          *prometheus.CounterVec
	shardDistributionHist *prometheus.HistogramVec
}

func NewMetrics(namespace string) *Metrics {
	return &Metrics{
		decoratedEvents: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "decorated_event_total",
				Help:      "The number of decorated events",
			},
			[]string{"event", "network"},
		),
		samplingProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_processed_total",
				Help:      "The number of messages processed after sampling",
			},
			[]string{"event_type"},
		),
		samplingSkipped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_skipped_total",
				Help:      "The number of messages skipped due to sampling",
			},
			[]string{"event_type"},
		),
		shardDistribution: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "shard_distribution_total",
				Help:      "The distribution of messages across shards",
			},
			[]string{"topic", "shard"},
		),
		shardProcessed: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "shard_processed_total",
				Help:      "The number of messages processed by shard",
			},
			[]string{"topic", "shard"},
		),
		shardSkipped: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "shard_skipped_total",
				Help:      "The number of messages skipped by shard",
			},
			[]string{"topic", "shard"},
		),
		shardDistributionHist: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "shard_distribution_histogram",
				Help:      "Histogram of shard distribution",
				Buckets:   prometheus.LinearBuckets(0, 1, 64), // 64 buckets, one per potential shard
			},
			[]string{"topic"},
		),
	}
}

func (m *Metrics) AddDecoratedEvent(count float64, eventType, network string) {
	m.decoratedEvents.WithLabelValues(eventType, network).Add(count)
}

func (m *Metrics) AddProcessedMessage(eventType string) {
	m.samplingProcessed.WithLabelValues(eventType).Inc()
}

func (m *Metrics) AddSkippedMessage(eventType string) {
	m.samplingSkipped.WithLabelValues(eventType).Inc()
}

// AddShardObservation records a message being assigned to a particular shard
func (m *Metrics) AddShardObservation(topic string, shard uint64) {
	shardStr := fmt.Sprintf("%d", shard)
	m.shardDistribution.WithLabelValues(topic, shardStr).Inc()
	m.shardDistributionHist.WithLabelValues(topic).Observe(float64(shard))
}

// AddShardProcessed records a message being processed from a particular shard
func (m *Metrics) AddShardProcessed(topic string, shard uint64) {
	shardStr := fmt.Sprintf("%d", shard)
	m.shardProcessed.WithLabelValues(topic, shardStr).Inc()
}

// AddShardSkipped records a message being skipped from a particular shard
func (m *Metrics) AddShardSkipped(topic string, shard uint64) {
	shardStr := fmt.Sprintf("%d", shard)
	m.shardSkipped.WithLabelValues(topic, shardStr).Inc()
}
