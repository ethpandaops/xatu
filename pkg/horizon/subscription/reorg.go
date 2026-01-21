package subscription

import (
	"context"
	"sync"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// ReorgEvent represents a parsed chain reorg event from the SSE stream.
type ReorgEvent struct {
	// Slot is the slot at which the reorg occurred.
	Slot phase0.Slot
	// Depth is the number of slots in the reorg.
	Depth uint64
	// OldHeadBlock is the block root of the old head.
	OldHeadBlock phase0.Root
	// NewHeadBlock is the block root of the new head.
	NewHeadBlock phase0.Root
	// OldHeadState is the state root of the old head.
	OldHeadState phase0.Root
	// NewHeadState is the state root of the new head.
	NewHeadState phase0.Root
	// Epoch is the epoch in which the reorg occurred.
	Epoch phase0.Epoch
	// ReceivedAt is the time when the event was received.
	ReceivedAt time.Time
	// NodeName is the name of the beacon node that received this event.
	NodeName string
}

// ReorgSubscription manages SSE subscriptions to chain reorg events across multiple beacon nodes.
type ReorgSubscription struct {
	log     logrus.FieldLogger
	pool    *ethereum.BeaconNodePool
	metrics *ReorgMetrics

	// events channel receives parsed reorg events.
	events chan ReorgEvent

	// done channel signals subscription shutdown.
	done chan struct{}
	wg   sync.WaitGroup

	// config holds reorg subscription configuration.
	config *ReorgConfig
}

// ReorgMetrics tracks chain reorg metrics.
type ReorgMetrics struct {
	reorgsTotal    *prometheus.CounterVec
	reorgDepth     *prometheus.HistogramVec
	reorgsIgnored  *prometheus.CounterVec
	lastReorgAt    *prometheus.GaugeVec
	lastReorgDepth *prometheus.GaugeVec
	lastReorgSlot  *prometheus.GaugeVec
}

// NewReorgMetrics creates metrics for chain reorg subscriptions.
func NewReorgMetrics(namespace string) *ReorgMetrics {
	m := &ReorgMetrics{
		reorgsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "reorg",
			Name:      "events_total",
			Help:      "Total number of chain reorg events received from beacon nodes",
		}, []string{"node", "network"}),

		reorgDepth: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "reorg",
			Name:      "depth",
			Help:      "Histogram of chain reorg depths in slots",
			Buckets:   []float64{1, 2, 3, 4, 5, 6, 8, 10, 12, 16, 24, 32, 48, 64},
		}, []string{"node", "network"}),

		reorgsIgnored: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "reorg",
			Name:      "ignored_total",
			Help:      "Total number of chain reorg events ignored (depth exceeds limit)",
		}, []string{"node", "network"}),

		lastReorgAt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "reorg",
			Name:      "last_event_at",
			Help:      "Unix timestamp of last chain reorg event per beacon node",
		}, []string{"node", "network"}),

		lastReorgDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "reorg",
			Name:      "last_depth",
			Help:      "Depth (in slots) of the last chain reorg event per beacon node",
		}, []string{"node", "network"}),

		lastReorgSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "reorg",
			Name:      "last_slot",
			Help:      "Slot of the last chain reorg event per beacon node",
		}, []string{"node", "network"}),
	}

	prometheus.MustRegister(
		m.reorgsTotal,
		m.reorgDepth,
		m.reorgsIgnored,
		m.lastReorgAt,
		m.lastReorgDepth,
		m.lastReorgSlot,
	)

	return m
}

// ReorgConfig holds configuration for the reorg subscription.
type ReorgConfig struct {
	// Enabled indicates if reorg handling is enabled.
	Enabled bool `yaml:"enabled" default:"true"`
	// MaxDepth is the maximum reorg depth to handle. Reorgs deeper than this are ignored.
	// Default: 64 slots
	MaxDepth uint64 `yaml:"maxDepth" default:"64"`
	// BufferSize is the size of the events channel buffer.
	// Default: 100
	BufferSize int `yaml:"bufferSize" default:"100"`
}

// Validate validates the configuration.
func (c *ReorgConfig) Validate() error {
	if c.MaxDepth == 0 {
		c.MaxDepth = 64
	}

	if c.BufferSize <= 0 {
		c.BufferSize = 100
	}

	return nil
}

// NewReorgSubscription creates a new ReorgSubscription.
func NewReorgSubscription(
	log logrus.FieldLogger,
	pool *ethereum.BeaconNodePool,
	config *ReorgConfig,
) *ReorgSubscription {
	if config == nil {
		config = &ReorgConfig{
			Enabled:    true,
			MaxDepth:   64,
			BufferSize: 100,
		}
	}

	if config.MaxDepth == 0 {
		config.MaxDepth = 64
	}

	if config.BufferSize <= 0 {
		config.BufferSize = 100
	}

	return &ReorgSubscription{
		log:     log.WithField("component", "subscription/reorg"),
		pool:    pool,
		metrics: NewReorgMetrics("xatu_horizon"),
		events:  make(chan ReorgEvent, config.BufferSize),
		done:    make(chan struct{}),
		config:  config,
	}
}

// Start starts subscribing to chain reorg events on all beacon nodes.
// This should be called after the beacon node pool is started and ready.
func (r *ReorgSubscription) Start(ctx context.Context) error {
	if !r.config.Enabled {
		r.log.Info("Reorg subscription disabled")

		return nil
	}

	r.log.Info("Starting reorg subscription")

	// Get all nodes from the pool and subscribe to each.
	nodes := r.pool.GetAllNodes()
	if len(nodes) == 0 {
		r.log.Warn("No beacon nodes configured for reorg subscription")

		return nil
	}

	for _, wrapper := range nodes {
		r.subscribeToNode(ctx, wrapper)
	}

	r.log.WithField("node_count", len(nodes)).Info("Reorg subscription started")

	return nil
}

// subscribeToNode subscribes to chain reorg events on a single beacon node.
func (r *ReorgSubscription) subscribeToNode(ctx context.Context, wrapper *ethereum.BeaconNodeWrapper) {
	nodeName := wrapper.Name()
	node := wrapper.Node()
	log := r.log.WithField("beacon_node", nodeName)

	// Get network name for metrics.
	networkName := "unknown"
	if metadata := r.pool.Metadata(); metadata != nil {
		networkName = string(metadata.Network.Name)
	}

	// Subscribe to chain reorg events.
	// The beacon library handles:
	// - SSE connection management
	// - Automatic reconnection with backoff
	// - Parsing of SSE payloads
	node.OnChainReOrg(ctx, func(ctx context.Context, event *eth2v1.ChainReorgEvent) error {
		receivedAt := time.Now()

		log.WithFields(logrus.Fields{
			"slot":           event.Slot,
			"depth":          event.Depth,
			"old_head_block": event.OldHeadBlock.String(),
			"new_head_block": event.NewHeadBlock.String(),
			"epoch":          event.Epoch,
		}).Info("Received chain reorg event")

		// Update metrics.
		r.metrics.reorgsTotal.WithLabelValues(nodeName, networkName).Inc()
		r.metrics.reorgDepth.WithLabelValues(nodeName, networkName).Observe(float64(event.Depth))
		r.metrics.lastReorgAt.WithLabelValues(nodeName, networkName).Set(float64(receivedAt.Unix()))
		r.metrics.lastReorgDepth.WithLabelValues(nodeName, networkName).Set(float64(event.Depth))
		r.metrics.lastReorgSlot.WithLabelValues(nodeName, networkName).Set(float64(event.Slot))

		// Check if depth exceeds limit.
		if event.Depth > r.config.MaxDepth {
			log.WithFields(logrus.Fields{
				"slot":      event.Slot,
				"depth":     event.Depth,
				"max_depth": r.config.MaxDepth,
			}).Warn("Ignoring reorg event - depth exceeds configured limit")

			r.metrics.reorgsIgnored.WithLabelValues(nodeName, networkName).Inc()

			return nil
		}

		// Create reorg event.
		reorgEvent := ReorgEvent{
			Slot:         event.Slot,
			Depth:        event.Depth,
			OldHeadBlock: event.OldHeadBlock,
			NewHeadBlock: event.NewHeadBlock,
			OldHeadState: event.OldHeadState,
			NewHeadState: event.NewHeadState,
			Epoch:        event.Epoch,
			ReceivedAt:   receivedAt,
			NodeName:     nodeName,
		}

		// Emit the reorg event to the channel.
		select {
		case r.events <- reorgEvent:
			// Event sent successfully.
		case <-ctx.Done():
			return ctx.Err()
		case <-r.done:
			return ErrSubscriptionClosed
		default:
			// Channel is full, log and drop the event.
			log.WithField("slot", event.Slot).Warn("Reorg event channel full, dropping event")
		}

		return nil
	})

	log.Debug("Subscribed to chain reorg events")
}

// Events returns the channel that receives reorg events.
// Consumers should read from this channel to process incoming reorg events.
func (r *ReorgSubscription) Events() <-chan ReorgEvent {
	return r.events
}

// Stop stops the reorg subscription.
func (r *ReorgSubscription) Stop(_ context.Context) error {
	r.log.Info("Stopping reorg subscription")

	close(r.done)
	r.wg.Wait()

	// Close events channel after all goroutines have stopped.
	close(r.events)

	r.log.Info("Reorg subscription stopped")

	return nil
}

// MaxDepth returns the configured maximum reorg depth.
func (r *ReorgSubscription) MaxDepth() uint64 {
	return r.config.MaxDepth
}

// Enabled returns whether reorg handling is enabled.
func (r *ReorgSubscription) Enabled() bool {
	return r.config.Enabled
}
