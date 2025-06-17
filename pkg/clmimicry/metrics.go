package clmimicry

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
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

	// Hierarchical sharding metrics.
	gossipTopicMatches  *prometheus.CounterVec
	fallbackUsage       *prometheus.CounterVec
	configTypeUsage     *prometheus.CounterVec
	gossipTopicSampling *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEvents: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events received",
		}, []string{"type", "network_id"}),
		samplingProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_processed_total",
				Help:      "The number of messages processed after sampling",
			},
			[]string{"type", "network_id"},
		),
		samplingSkipped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sampling_skipped_total",
				Help:      "The number of messages skipped due to sampling",
			},
			[]string{"type", "network_id"},
		),
		shardDistribution: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "shard_distribution_total",
				Help:      "The distribution of messages across shards",
			},
			[]string{"topic", "shard", "network_id"},
		),
		shardProcessed: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "shard_processed_total",
				Help:      "The number of messages processed by shard",
			},
			[]string{"topic", "shard", "network_id"},
		),
		shardSkipped: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "shard_skipped_total",
				Help:      "The number of messages skipped by shard",
			},
			[]string{"topic", "shard", "network_id"},
		),
		shardDistributionHist: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: namespace,
				Name:      "shard_distribution_histogram",
				Help:      "Histogram of shard distribution",
				Buckets:   prometheus.LinearBuckets(0, 1, 64), // 64 buckets, one per potential shard
			},
			[]string{"topic", "network_id"},
		),
		gossipTopicMatches: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gossip_topic_pattern_matches_total",
				Help:      "Number of times each gossip topic pattern was matched",
			},
			[]string{"event_type", "gossip_pattern", "gossip_topic", "network_id"},
		),
		fallbackUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "hierarchical_fallback_usage_total",
				Help:      "Number of times hierarchical fallback configuration was used",
			},
			[]string{"event_type", "reason", "network_id"}, // reason: no_topic, no_match
		),
		configTypeUsage: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "sharding_config_type_usage_total",
				Help:      "Number of times each configuration type was used",
			},
			[]string{"event_type", "config_type", "network_id"}, // config_type: simple, hierarchical
		),
		gossipTopicSampling: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: namespace,
				Name:      "gossip_topic_sampling_total",
				Help:      "Sampling statistics per gossip topic",
			},
			[]string{"event_type", "gossip_topic", "action", "network_id"}, // action: processed, skipped
		),
	}

	prometheus.MustRegister(m.decoratedEvents)
	prometheus.MustRegister(m.samplingProcessed)
	prometheus.MustRegister(m.samplingSkipped)
	prometheus.MustRegister(m.shardDistribution)
	prometheus.MustRegister(m.shardProcessed)
	prometheus.MustRegister(m.shardSkipped)
	prometheus.MustRegister(m.shardDistributionHist)
	prometheus.MustRegister(m.gossipTopicMatches)
	prometheus.MustRegister(m.fallbackUsage)
	prometheus.MustRegister(m.configTypeUsage)
	prometheus.MustRegister(m.gossipTopicSampling)

	return m
}

func (m *Metrics) AddDecoratedEvent(count float64, eventType, network string) {
	m.decoratedEvents.WithLabelValues(eventType, network).Add(count)
}

func (m *Metrics) AddProcessedMessage(eventType, network string) {
	m.samplingProcessed.WithLabelValues(eventType, network).Inc()
}

func (m *Metrics) AddSkippedMessage(eventType, network string) {
	m.samplingSkipped.WithLabelValues(eventType, network).Inc()
}

// AddShardObservation records a message being assigned to a particular shard
func (m *Metrics) AddShardObservation(topic string, shard uint64, network string) {
	shardStr := fmt.Sprintf("%d", shard)
	m.shardDistribution.WithLabelValues(topic, shardStr, network).Inc()
	m.shardDistributionHist.WithLabelValues(topic, network).Observe(float64(shard))
}

// AddShardProcessed records a message being processed from a particular shard
func (m *Metrics) AddShardProcessed(topic string, shard uint64, network string) {
	shardStr := fmt.Sprintf("%d", shard)
	m.shardProcessed.WithLabelValues(topic, shardStr, network).Inc()
}

// AddShardSkipped records a message being skipped from a particular shard
func (m *Metrics) AddShardSkipped(topic string, shard uint64, network string) {
	shardStr := fmt.Sprintf("%d", shard)
	m.shardSkipped.WithLabelValues(topic, shardStr, network).Inc()
}

// AddGossipTopicMatch records a successful gossip topic pattern match
func (m *Metrics) AddGossipTopicMatch(eventType, gossipPattern, gossipTopic, network string) {
	m.gossipTopicMatches.WithLabelValues(eventType, gossipPattern, gossipTopic, network).Inc()
}

// AddFallbackUsage records when hierarchical fallback configuration is used
func (m *Metrics) AddFallbackUsage(eventType, reason, network string) {
	m.fallbackUsage.WithLabelValues(eventType, reason, network).Inc()
}

// AddConfigTypeUsage records usage of different configuration types
func (m *Metrics) AddConfigTypeUsage(eventType, configType, network string) {
	m.configTypeUsage.WithLabelValues(eventType, configType, network).Inc()
}

// AddGossipTopicSampling records sampling actions per gossip topic
func (m *Metrics) AddGossipTopicSampling(eventType, gossipTopic, action, network string) {
	m.gossipTopicSampling.WithLabelValues(eventType, gossipTopic, action, network).Inc()
}
