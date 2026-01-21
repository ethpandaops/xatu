package iterator

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/horizon/cache"
	"github.com/ethpandaops/xatu/pkg/horizon/coordinator"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/ethpandaops/xatu/pkg/horizon/subscription"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	cldataIterator "github.com/ethpandaops/xatu/pkg/cldata/iterator"
)

var (
	// ErrIteratorClosed is returned when the iterator is closed.
	ErrIteratorClosed = errors.New("iterator closed")
	// ErrSlotSkipped is returned when a slot should be skipped (not an error condition).
	ErrSlotSkipped = errors.New("slot skipped")
)

// HeadIteratorConfig holds configuration for the HEAD iterator.
type HeadIteratorConfig struct {
	// Enabled indicates if this iterator is enabled.
	Enabled bool `yaml:"enabled" default:"true"`
}

// Validate validates the configuration.
func (c *HeadIteratorConfig) Validate() error {
	return nil
}

// HeadIterator is an iterator that tracks the HEAD of the beacon chain.
// It receives real-time block events from SSE subscriptions and processes
// them in order, coordinating with the server to track progress.
type HeadIterator struct {
	log         logrus.FieldLogger
	pool        *ethereum.BeaconNodePool
	coordinator *coordinator.Client
	dedupCache  *cache.DedupCache
	metrics     *HeadIteratorMetrics

	// horizonType is the type of deriver this iterator is for.
	horizonType xatu.HorizonType
	// networkID is the network identifier.
	networkID string
	// networkName is the human-readable network name.
	networkName string

	// blockEvents receives deduplicated block events from SSE.
	blockEvents <-chan subscription.BlockEvent

	// activationFork is the fork at which the deriver becomes active.
	activationFork spec.DataVersion

	// currentPosition tracks the last processed position.
	currentPosition *cldataIterator.Position
	positionMu      sync.RWMutex

	// done signals iterator shutdown.
	done chan struct{}
}

// HeadIteratorMetrics tracks metrics for the HEAD iterator.
type HeadIteratorMetrics struct {
	processedTotal   *prometheus.CounterVec
	skippedTotal     *prometheus.CounterVec
	lastProcessedAt  *prometheus.GaugeVec
	positionSlot     *prometheus.GaugeVec
	eventsQueuedSize prometheus.Gauge
}

// NewHeadIteratorMetrics creates new metrics for the HEAD iterator.
func NewHeadIteratorMetrics(namespace string) *HeadIteratorMetrics {
	m := &HeadIteratorMetrics{
		processedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "head_iterator",
			Name:      "processed_total",
			Help:      "Total number of slots processed by the HEAD iterator",
		}, []string{"deriver", "network"}),

		skippedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "head_iterator",
			Name:      "skipped_total",
			Help:      "Total number of slots skipped (already processed)",
		}, []string{"deriver", "network", "reason"}),

		lastProcessedAt: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "head_iterator",
			Name:      "last_processed_at",
			Help:      "Unix timestamp of last processed slot",
		}, []string{"deriver", "network"}),

		positionSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "head_iterator",
			Name:      "position_slot",
			Help:      "Current slot position of the HEAD iterator",
		}, []string{"deriver", "network"}),

		eventsQueuedSize: prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "head_iterator",
			Name:      "events_queued",
			Help:      "Number of block events queued for processing",
		}),
	}

	prometheus.MustRegister(
		m.processedTotal,
		m.skippedTotal,
		m.lastProcessedAt,
		m.positionSlot,
		m.eventsQueuedSize,
	)

	return m
}

// NewHeadIterator creates a new HEAD iterator.
func NewHeadIterator(
	log logrus.FieldLogger,
	pool *ethereum.BeaconNodePool,
	coordinatorClient *coordinator.Client,
	dedupCache *cache.DedupCache,
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
	blockEvents <-chan subscription.BlockEvent,
) *HeadIterator {
	return &HeadIterator{
		log: log.WithFields(logrus.Fields{
			"component":    "iterator/head",
			"horizon_type": horizonType.String(),
		}),
		pool:        pool,
		coordinator: coordinatorClient,
		dedupCache:  dedupCache,
		horizonType: horizonType,
		networkID:   networkID,
		networkName: networkName,
		blockEvents: blockEvents,
		metrics:     NewHeadIteratorMetrics("xatu_horizon"),
		done:        make(chan struct{}),
	}
}

// Start initializes the iterator with the activation fork version.
func (h *HeadIterator) Start(_ context.Context, activationFork spec.DataVersion) error {
	h.activationFork = activationFork

	h.log.WithFields(logrus.Fields{
		"activation_fork": activationFork.String(),
		"network_id":      h.networkID,
	}).Info("HEAD iterator started")

	return nil
}

// Next returns the next position to process.
// It blocks until a block event is received from the SSE subscription,
// then returns the slot for processing.
func (h *HeadIterator) Next(ctx context.Context) (*cldataIterator.Position, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-h.done:
			return nil, ErrIteratorClosed
		case event, ok := <-h.blockEvents:
			if !ok {
				return nil, ErrIteratorClosed
			}

			// Check if we should process this slot.
			position, err := h.processBlockEvent(ctx, &event)
			if err != nil {
				if errors.Is(err, ErrSlotSkipped) {
					// Slot was skipped (duplicate or already processed), continue to next.
					continue
				}

				h.log.WithError(err).WithField("slot", event.Slot).
					Warn("Failed to process block event")

				continue
			}

			return position, nil
		}
	}
}

// processBlockEvent processes a block event and returns a position if it should be processed.
// Returns ErrSlotSkipped if the slot should be skipped (not an error condition).
func (h *HeadIterator) processBlockEvent(ctx context.Context, event *subscription.BlockEvent) (*cldataIterator.Position, error) {
	// Check deduplication cache first.
	blockRootStr := event.BlockRoot.String()
	if h.dedupCache.Check(blockRootStr) {
		// This block root was already seen, skip it.
		h.metrics.skippedTotal.WithLabelValues(
			h.horizonType.String(),
			h.networkName,
			"duplicate",
		).Inc()

		h.log.WithFields(logrus.Fields{
			"slot":       event.Slot,
			"block_root": blockRootStr,
		}).Trace("Skipping duplicate block event")

		return nil, ErrSlotSkipped
	}

	// Check if we need to skip based on activation fork.
	if err := h.checkActivationFork(event.Slot); err != nil {
		h.metrics.skippedTotal.WithLabelValues(
			h.horizonType.String(),
			h.networkName,
			"pre_activation",
		).Inc()

		h.log.WithFields(logrus.Fields{
			"slot":       event.Slot,
			"block_root": blockRootStr,
			"reason":     err.Error(),
		}).Trace("Skipping block event due to activation fork")

		return nil, ErrSlotSkipped
	}

	// Check coordinator to see if this slot was already processed.
	alreadyProcessed, err := h.isSlotProcessed(ctx, event.Slot)
	if err != nil {
		return nil, fmt.Errorf("failed to check if slot is processed: %w", err)
	}

	if alreadyProcessed {
		h.metrics.skippedTotal.WithLabelValues(
			h.horizonType.String(),
			h.networkName,
			"already_processed",
		).Inc()

		h.log.WithFields(logrus.Fields{
			"slot":       event.Slot,
			"block_root": blockRootStr,
		}).Trace("Skipping already processed slot")

		return nil, ErrSlotSkipped
	}

	// Create position for the slot.
	position := &cldataIterator.Position{
		Slot:      event.Slot,
		Epoch:     phase0.Epoch(uint64(event.Slot) / 32), // Assumes 32 slots per epoch.
		Direction: cldataIterator.DirectionForward,
	}

	h.log.WithFields(logrus.Fields{
		"slot":       event.Slot,
		"block_root": blockRootStr,
		"epoch":      position.Epoch,
	}).Debug("Processing block event")

	return position, nil
}

// checkActivationFork checks if the slot is after the activation fork.
func (h *HeadIterator) checkActivationFork(slot phase0.Slot) error {
	// Phase0 is always active.
	if h.activationFork == spec.DataVersionPhase0 {
		return nil
	}

	metadata := h.pool.Metadata()
	if metadata == nil {
		return errors.New("metadata not available")
	}

	beaconSpec := metadata.Spec
	if beaconSpec == nil {
		return errors.New("spec not available")
	}

	forkEpoch, err := beaconSpec.ForkEpochs.GetByName(h.activationFork.String())
	if err != nil {
		return fmt.Errorf("failed to get fork epoch for %s: %w", h.activationFork.String(), err)
	}

	slotsPerEpoch := uint64(beaconSpec.SlotsPerEpoch)
	forkSlot := phase0.Slot(uint64(forkEpoch.Epoch) * slotsPerEpoch)

	if slot < forkSlot {
		return fmt.Errorf("slot %d is before fork activation at slot %d", slot, forkSlot)
	}

	return nil
}

// isSlotProcessed checks if a slot has already been processed by either HEAD or FILL iterator.
// Both iterators coordinate through the coordinator service:
// - HEAD updates head_slot after processing real-time blocks
// - FILL updates fill_slot after processing historical slots
// A slot is considered processed if slot <= head_slot OR slot <= fill_slot.
func (h *HeadIterator) isSlotProcessed(ctx context.Context, slot phase0.Slot) (bool, error) {
	location, err := h.coordinator.GetHorizonLocation(ctx, h.horizonType, h.networkID)
	if err != nil {
		// If location doesn't exist, the slot hasn't been processed.
		// Check if it's a "not found" error and return false.
		// Otherwise, return the error.
		// Note: The coordinator client should return nil location for not found.
		return false, fmt.Errorf("failed to get horizon location: %w", err)
	}

	if location == nil {
		// No location stored yet, nothing has been processed.
		return false, nil
	}

	// Check if this slot was processed by HEAD (slot <= head_slot)
	// or by FILL (slot <= fill_slot).
	// Both iterators skip slots processed by the other to avoid duplicates.
	if uint64(slot) <= location.HeadSlot {
		return true, nil
	}

	if uint64(slot) <= location.FillSlot {
		return true, nil
	}

	return false, nil
}

// UpdateLocation persists the current position after successful processing.
func (h *HeadIterator) UpdateLocation(ctx context.Context, position *cldataIterator.Position) error {
	// Get current location from coordinator.
	location, err := h.coordinator.GetHorizonLocation(ctx, h.horizonType, h.networkID)
	if err != nil {
		// Treat as new location if not found.
		location = nil
	}

	// Create or update the location.
	var headSlot uint64

	var fillSlot uint64

	if location != nil {
		fillSlot = location.FillSlot

		// Only update head_slot if the new position is greater.
		headSlot = max(uint64(position.Slot), location.HeadSlot)
	} else {
		// New location - initialize both to current slot.
		headSlot = uint64(position.Slot)
		fillSlot = uint64(position.Slot)
	}

	newLocation := &xatu.HorizonLocation{
		NetworkId: h.networkID,
		Type:      h.horizonType,
		HeadSlot:  headSlot,
		FillSlot:  fillSlot,
	}

	if err := h.coordinator.UpsertHorizonLocation(ctx, newLocation); err != nil {
		return fmt.Errorf("failed to upsert horizon location: %w", err)
	}

	// Update current position.
	h.positionMu.Lock()
	h.currentPosition = position
	h.positionMu.Unlock()

	// Update metrics.
	h.metrics.processedTotal.WithLabelValues(
		h.horizonType.String(),
		h.networkName,
	).Inc()
	h.metrics.positionSlot.WithLabelValues(
		h.horizonType.String(),
		h.networkName,
	).Set(float64(position.Slot))

	h.log.WithFields(logrus.Fields{
		"slot":      position.Slot,
		"head_slot": headSlot,
		"fill_slot": fillSlot,
	}).Debug("Updated horizon location")

	return nil
}

// Stop stops the HEAD iterator.
func (h *HeadIterator) Stop(_ context.Context) error {
	close(h.done)

	h.log.Info("HEAD iterator stopped")

	return nil
}

// CurrentPosition returns the current position of the iterator.
func (h *HeadIterator) CurrentPosition() *cldataIterator.Position {
	h.positionMu.RLock()
	defer h.positionMu.RUnlock()

	return h.currentPosition
}

// HorizonType returns the horizon type this iterator is for.
func (h *HeadIterator) HorizonType() xatu.HorizonType {
	return h.horizonType
}

// Verify HeadIterator implements the Iterator interface.
var _ cldataIterator.Iterator = (*HeadIterator)(nil)
