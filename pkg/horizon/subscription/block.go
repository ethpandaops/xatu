package subscription

import (
	"context"
	"errors"
	"sync"
	"time"

	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var ErrSubscriptionClosed = errors.New("subscription closed")

// BlockEvent represents a parsed block event from the SSE stream.
type BlockEvent struct {
	// Slot is the slot number of the block.
	Slot phase0.Slot
	// BlockRoot is the root of the block.
	BlockRoot phase0.Root
	// ExecutionOptimistic indicates if the block was received before execution validation.
	ExecutionOptimistic bool
	// ReceivedAt is the time when the event was received.
	ReceivedAt time.Time
	// NodeName is the name of the beacon node that received this event.
	NodeName string
}

// BlockSubscription manages SSE subscriptions to block events across multiple beacon nodes.
type BlockSubscription struct {
	log     logrus.FieldLogger
	pool    *ethereum.BeaconNodePool
	metrics *Metrics

	// events channel receives parsed block events.
	events chan BlockEvent

	// done channel signals subscription shutdown.
	done chan struct{}
	wg   sync.WaitGroup

	// bufferSize is the size of the events channel buffer.
	bufferSize int
}

// Metrics tracks SSE subscription metrics.
type Metrics struct {
	sseEventsTotal          *prometheus.CounterVec
	sseConnectionStatus     *prometheus.GaugeVec
	sseReconnectsTotal      *prometheus.CounterVec
	sseLastEventReceivedAt  *prometheus.GaugeVec
	sseEventProcessingDelay *prometheus.HistogramVec
}

// NewMetrics creates metrics for SSE subscriptions.
func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		sseEventsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sse",
			Name:      "events_total",
			Help:      "Total number of SSE events received from beacon nodes",
		}, []string{"node", "topic", "network"}),

		sseConnectionStatus: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "sse",
			Name:      "connection_status",
			Help:      "SSE connection status per beacon node (1=connected, 0=disconnected)",
		}, []string{"node"}),

		sseReconnectsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "sse",
			Name:      "reconnects_total",
			Help:      "Total number of SSE reconnection attempts per beacon node",
		}, []string{"node"}),

		sseLastEventReceivedAt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "sse",
			Name:      "last_event_received_at",
			Help:      "Unix timestamp of last SSE event received per beacon node",
		}, []string{"node", "topic"}),

		sseEventProcessingDelay: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: "sse",
			Name:      "event_processing_delay_seconds",
			Help:      "Time delay between slot start and event receipt",
			Buckets:   []float64{0.1, 0.25, 0.5, 1.0, 2.0, 4.0, 8.0, 12.0},
		}, []string{"node", "topic"}),
	}

	prometheus.MustRegister(
		m.sseEventsTotal,
		m.sseConnectionStatus,
		m.sseReconnectsTotal,
		m.sseLastEventReceivedAt,
		m.sseEventProcessingDelay,
	)

	return m
}

// IncSSEEvents increments the SSE events counter.
func (m *Metrics) IncSSEEvents(node, topic, network string) {
	m.sseEventsTotal.WithLabelValues(node, topic, network).Inc()
}

// SetSSEConnectionStatus sets the SSE connection status for a node.
func (m *Metrics) SetSSEConnectionStatus(node string, connected bool) {
	val := float64(0)
	if connected {
		val = 1
	}

	m.sseConnectionStatus.WithLabelValues(node).Set(val)
}

// IncSSEReconnects increments the SSE reconnect counter.
func (m *Metrics) IncSSEReconnects(node string) {
	m.sseReconnectsTotal.WithLabelValues(node).Inc()
}

// SetSSELastEventReceivedAt sets the timestamp of the last received event.
func (m *Metrics) SetSSELastEventReceivedAt(node, topic string, t time.Time) {
	m.sseLastEventReceivedAt.WithLabelValues(node, topic).Set(float64(t.Unix()))
}

// ObserveSSEEventProcessingDelay records the processing delay for an event.
func (m *Metrics) ObserveSSEEventProcessingDelay(node, topic string, delay time.Duration) {
	m.sseEventProcessingDelay.WithLabelValues(node, topic).Observe(delay.Seconds())
}

// Config holds configuration for the block subscription.
type Config struct {
	// BufferSize is the size of the events channel buffer.
	// Default: 1000
	BufferSize int `yaml:"bufferSize" default:"1000"`
}

// Validate validates the configuration.
func (c *Config) Validate() error {
	if c.BufferSize <= 0 {
		c.BufferSize = 1000
	}

	return nil
}

// NewBlockSubscription creates a new BlockSubscription.
func NewBlockSubscription(
	log logrus.FieldLogger,
	pool *ethereum.BeaconNodePool,
	config *Config,
) *BlockSubscription {
	if config == nil {
		config = &Config{BufferSize: 1000}
	}

	if config.BufferSize <= 0 {
		config.BufferSize = 1000
	}

	return &BlockSubscription{
		log:        log.WithField("component", "subscription/block"),
		pool:       pool,
		metrics:    NewMetrics("xatu_horizon"),
		events:     make(chan BlockEvent, config.BufferSize),
		done:       make(chan struct{}),
		bufferSize: config.BufferSize,
	}
}

// Start starts subscribing to block events on all beacon nodes.
// This should be called after the beacon node pool is started and ready.
func (b *BlockSubscription) Start(ctx context.Context) error {
	b.log.Info("Starting block subscription")

	// Get all nodes from the pool and subscribe to each.
	nodes := b.pool.GetAllNodes()
	if len(nodes) == 0 {
		return errors.New("no beacon nodes configured")
	}

	for _, wrapper := range nodes {
		b.subscribeToNode(ctx, wrapper)
	}

	b.log.WithField("node_count", len(nodes)).Info("Block subscription started")

	return nil
}

// subscribeToNode subscribes to block events on a single beacon node.
func (b *BlockSubscription) subscribeToNode(ctx context.Context, wrapper *ethereum.BeaconNodeWrapper) {
	nodeName := wrapper.Name()
	node := wrapper.Node()
	log := b.log.WithField("beacon_node", nodeName)

	// Get network name for metrics.
	networkName := "unknown"
	if metadata := b.pool.Metadata(); metadata != nil {
		networkName = string(metadata.Network.Name)
	}

	// Subscribe to block events.
	// The beacon library handles:
	// - SSE connection management
	// - Automatic reconnection with backoff
	// - Parsing of SSE payloads
	node.OnBlock(ctx, func(ctx context.Context, event *eth2v1.BlockEvent) error {
		receivedAt := time.Now()

		log.WithFields(logrus.Fields{
			"slot":                 event.Slot,
			"block_root":           event.Block.String(),
			"execution_optimistic": event.ExecutionOptimistic,
		}).Trace("Received block event")

		// Update metrics.
		b.metrics.IncSSEEvents(nodeName, "block", networkName)
		b.metrics.SetSSELastEventReceivedAt(nodeName, "block", receivedAt)

		// Calculate processing delay if we have wallclock.
		if metadata := b.pool.Metadata(); metadata != nil && metadata.Wallclock() != nil {
			slotInfo := metadata.Wallclock().Slots().FromNumber(uint64(event.Slot))
			delay := receivedAt.Sub(slotInfo.TimeWindow().Start())
			b.metrics.ObserveSSEEventProcessingDelay(nodeName, "block", delay)
		}

		// Emit the block event to the channel.
		blockEvent := BlockEvent{
			Slot:                event.Slot,
			BlockRoot:           event.Block,
			ExecutionOptimistic: event.ExecutionOptimistic,
			ReceivedAt:          receivedAt,
			NodeName:            nodeName,
		}

		select {
		case b.events <- blockEvent:
			// Event sent successfully.
		case <-ctx.Done():
			return ctx.Err()
		case <-b.done:
			return ErrSubscriptionClosed
		default:
			// Channel is full, log and drop the event.
			log.WithField("slot", event.Slot).Warn("Block event channel full, dropping event")
		}

		return nil
	})

	// Subscribe to connection events for status tracking.
	// The beacon library emits these when connection state changes.
	node.OnFirstTimeHealthy(ctx, func(_ context.Context, _ *beacon.FirstTimeHealthyEvent) error {
		log.Debug("Beacon node SSE connection established")
		b.metrics.SetSSEConnectionStatus(nodeName, true)

		return nil
	})

	log.Debug("Subscribed to block events")
}

// Events returns the channel that receives block events.
// Consumers should read from this channel to process incoming block events.
func (b *BlockSubscription) Events() <-chan BlockEvent {
	return b.events
}

// Stop stops the block subscription.
func (b *BlockSubscription) Stop(_ context.Context) error {
	b.log.Info("Stopping block subscription")

	close(b.done)
	b.wg.Wait()

	// Close events channel after all goroutines have stopped.
	close(b.events)

	b.log.Info("Block subscription stopped")

	return nil
}
