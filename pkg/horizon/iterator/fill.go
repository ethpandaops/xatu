package iterator

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/horizon/coordinator"
	"github.com/ethpandaops/xatu/pkg/horizon/ethereum"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	cldataIterator "github.com/ethpandaops/xatu/pkg/cldata/iterator"
)

const (
	// DefaultLagSlots is the default number of slots to stay behind HEAD.
	DefaultLagSlots = 32
	// DefaultMaxBoundedSlots is the default maximum range of slots to process per cycle.
	DefaultMaxBoundedSlots = 7200
	// DefaultFillRateLimit is the default rate limit in slots per second.
	DefaultFillRateLimit = 10.0
)

// FillIteratorConfig holds configuration for the FILL iterator.
type FillIteratorConfig struct {
	// Enabled indicates if this iterator is enabled.
	Enabled bool `yaml:"enabled" default:"true"`
	// LagSlots is the number of slots to stay behind HEAD.
	LagSlots uint64 `yaml:"lagSlots" default:"32"`
	// MaxBoundedSlots is the maximum number of slots to process in one bounded range.
	MaxBoundedSlots uint64 `yaml:"maxBoundedSlots" default:"7200"`
	// RateLimit is the maximum number of slots to process per second.
	RateLimit float64 `yaml:"rateLimit" default:"10.0"`
}

// Validate validates the configuration.
func (c *FillIteratorConfig) Validate() error {
	if c.LagSlots == 0 {
		c.LagSlots = DefaultLagSlots
	}

	if c.MaxBoundedSlots == 0 {
		c.MaxBoundedSlots = DefaultMaxBoundedSlots
	}

	if c.RateLimit <= 0 {
		c.RateLimit = DefaultFillRateLimit
	}

	return nil
}

// FillIterator is an iterator that fills in gaps by walking slots from fill_slot toward HEAD - LAG.
// It processes historical slots that may have been missed by the HEAD iterator.
type FillIterator struct {
	log         logrus.FieldLogger
	pool        *ethereum.BeaconNodePool
	coordinator *coordinator.Client
	config      *FillIteratorConfig
	metrics     *FillIteratorMetrics

	// horizonType is the type of deriver this iterator is for.
	horizonType xatu.HorizonType
	// networkID is the network identifier.
	networkID string
	// networkName is the human-readable network name.
	networkName string

	// activationFork is the fork at which the deriver becomes active.
	activationFork spec.DataVersion

	// currentSlot tracks the current slot being processed.
	currentSlot   phase0.Slot
	currentSlotMu sync.RWMutex

	// limiter controls the rate of slot processing.
	limiter *rate.Limiter

	// done signals iterator shutdown.
	done chan struct{}

	// started indicates if the iterator has been started.
	started bool
}

// FillIteratorMetrics tracks metrics for the FILL iterator.
type FillIteratorMetrics struct {
	processedTotal      *prometheus.CounterVec
	skippedTotal        *prometheus.CounterVec
	positionSlot        *prometheus.GaugeVec
	targetSlot          *prometheus.GaugeVec
	slotsRemaining      *prometheus.GaugeVec
	rateLimitWaitTotal  prometheus.Counter
	cyclesCompleteTotal *prometheus.CounterVec
}

// NewFillIteratorMetrics creates new metrics for the FILL iterator.
func NewFillIteratorMetrics(namespace string) *FillIteratorMetrics {
	m := &FillIteratorMetrics{
		processedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "processed_total",
			Help:      "Total number of slots processed by the FILL iterator",
		}, []string{"deriver", "network"}),

		skippedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "skipped_total",
			Help:      "Total number of slots skipped by the FILL iterator",
		}, []string{"deriver", "network", "reason"}),

		positionSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "position_slot",
			Help:      "Current slot position of the FILL iterator",
		}, []string{"deriver", "network"}),

		targetSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "target_slot",
			Help:      "Target slot the FILL iterator is working toward (HEAD - LAG)",
		}, []string{"deriver", "network"}),

		slotsRemaining: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "slots_remaining",
			Help:      "Number of slots remaining until caught up with target",
		}, []string{"deriver", "network"}),

		rateLimitWaitTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "rate_limit_wait_total",
			Help:      "Total number of times the FILL iterator waited for rate limit",
		}),

		cyclesCompleteTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "fill_iterator",
			Name:      "cycles_complete_total",
			Help:      "Total number of fill cycles completed (caught up to target)",
		}, []string{"deriver", "network"}),
	}

	prometheus.MustRegister(
		m.processedTotal,
		m.skippedTotal,
		m.positionSlot,
		m.targetSlot,
		m.slotsRemaining,
		m.rateLimitWaitTotal,
		m.cyclesCompleteTotal,
	)

	return m
}

// NewFillIterator creates a new FILL iterator.
func NewFillIterator(
	log logrus.FieldLogger,
	pool *ethereum.BeaconNodePool,
	coordinatorClient *coordinator.Client,
	config *FillIteratorConfig,
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
) *FillIterator {
	if config == nil {
		config = &FillIteratorConfig{}
	}

	_ = config.Validate()

	return &FillIterator{
		log: log.WithFields(logrus.Fields{
			"component":    "iterator/fill",
			"horizon_type": horizonType.String(),
		}),
		pool:        pool,
		coordinator: coordinatorClient,
		config:      config,
		horizonType: horizonType,
		networkID:   networkID,
		networkName: networkName,
		limiter:     rate.NewLimiter(rate.Limit(config.RateLimit), 1),
		metrics:     NewFillIteratorMetrics("xatu_horizon"),
		done:        make(chan struct{}),
	}
}

// Start initializes the iterator with the activation fork version.
func (f *FillIterator) Start(ctx context.Context, activationFork spec.DataVersion) error {
	f.activationFork = activationFork

	// Initialize current slot from coordinator
	location, err := f.coordinator.GetHorizonLocation(ctx, f.horizonType, f.networkID)
	if err != nil {
		// If location doesn't exist, we'll start from activation fork slot
		f.log.WithError(err).Debug("No existing location found, will start from activation fork")

		location = nil
	}

	if location != nil && location.FillSlot > 0 {
		f.currentSlot = phase0.Slot(location.FillSlot)
	} else {
		// Start from activation fork slot
		activationSlot, err := f.getActivationSlot()
		if err != nil {
			return fmt.Errorf("failed to get activation slot: %w", err)
		}

		f.currentSlot = activationSlot
	}

	f.started = true

	f.log.WithFields(logrus.Fields{
		"activation_fork": activationFork.String(),
		"network_id":      f.networkID,
		"start_slot":      f.currentSlot,
		"lag_slots":       f.config.LagSlots,
		"rate_limit":      f.config.RateLimit,
	}).Info("FILL iterator started")

	return nil
}

// Next returns the next position to process.
// It walks slots forward from fill_slot toward HEAD - LAG.
func (f *FillIterator) Next(ctx context.Context) (*cldataIterator.Position, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-f.done:
			return nil, ErrIteratorClosed
		default:
		}

		// Get target slot (HEAD - LAG)
		targetSlot, err := f.getTargetSlot(ctx)
		if err != nil {
			f.log.WithError(err).Warn("Failed to get target slot, will retry")
			time.Sleep(time.Second)

			continue
		}

		f.currentSlotMu.RLock()
		currentSlot := f.currentSlot
		f.currentSlotMu.RUnlock()

		// Update metrics
		f.metrics.targetSlot.WithLabelValues(f.horizonType.String(), f.networkName).
			Set(float64(targetSlot))
		f.metrics.positionSlot.WithLabelValues(f.horizonType.String(), f.networkName).
			Set(float64(currentSlot))

		if currentSlot < targetSlot {
			remaining := uint64(targetSlot) - uint64(currentSlot)
			f.metrics.slotsRemaining.WithLabelValues(f.horizonType.String(), f.networkName).
				Set(float64(remaining))
		} else {
			f.metrics.slotsRemaining.WithLabelValues(f.horizonType.String(), f.networkName).
				Set(0)
		}

		// Check if we've caught up to target
		if currentSlot >= targetSlot {
			f.metrics.cyclesCompleteTotal.WithLabelValues(f.horizonType.String(), f.networkName).Inc()

			f.log.WithFields(logrus.Fields{
				"current_slot": currentSlot,
				"target_slot":  targetSlot,
			}).Debug("FILL iterator caught up to target, waiting for new slots")

			// Wait before checking again
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-f.done:
				return nil, ErrIteratorClosed
			case <-time.After(time.Duration(12) * time.Second): // Wait roughly one slot
				continue
			}
		}

		// Apply rate limiting
		if err := f.limiter.Wait(ctx); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil, err
			}

			f.log.WithError(err).Warn("Rate limiter wait failed")

			continue
		}

		f.metrics.rateLimitWaitTotal.Inc()

		// Check if slot is before activation fork
		if err := f.checkActivationFork(currentSlot); err != nil {
			f.metrics.skippedTotal.WithLabelValues(
				f.horizonType.String(),
				f.networkName,
				"pre_activation",
			).Inc()

			f.log.WithFields(logrus.Fields{
				"slot":   currentSlot,
				"reason": err.Error(),
			}).Trace("Skipping slot due to activation fork")

			// Move to next slot
			f.incrementCurrentSlot()

			continue
		}

		// Apply bounded range limit
		if f.config.MaxBoundedSlots > 0 && currentSlot+phase0.Slot(f.config.MaxBoundedSlots) < targetSlot {
			// We're too far behind, jump forward
			newSlot := phase0.Slot(uint64(targetSlot) - f.config.MaxBoundedSlots)

			f.log.WithFields(logrus.Fields{
				"current_slot": currentSlot,
				"new_slot":     newSlot,
				"target_slot":  targetSlot,
				"max_bounded":  f.config.MaxBoundedSlots,
			}).Info("FILL iterator jumping forward due to bounded range limit")

			currentSlot = f.setCurrentSlot(newSlot)
		}

		// Create position for the slot
		position := &cldataIterator.Position{
			Slot:      currentSlot,
			Epoch:     phase0.Epoch(uint64(currentSlot) / 32), // Assumes 32 slots per epoch
			Direction: cldataIterator.DirectionBackward,       // FILL processes historical data
		}

		f.log.WithFields(logrus.Fields{
			"slot":        currentSlot,
			"epoch":       position.Epoch,
			"target_slot": targetSlot,
		}).Debug("Processing fill slot")

		// Advance current slot for next iteration
		f.incrementCurrentSlot()

		return position, nil
	}
}

// getTargetSlot returns the target slot (HEAD - LAG).
func (f *FillIterator) getTargetSlot(ctx context.Context) (phase0.Slot, error) {
	// Get current head slot from coordinator
	location, err := f.coordinator.GetHorizonLocation(ctx, f.horizonType, f.networkID)
	if err != nil {
		return 0, fmt.Errorf("failed to get horizon location: %w", err)
	}

	if location == nil || location.HeadSlot == 0 {
		// No head slot recorded yet, use wallclock
		return f.getWallclockHeadSlot()
	}

	headSlot := phase0.Slot(location.HeadSlot)

	// Calculate target: HEAD - LAG
	if uint64(headSlot) <= f.config.LagSlots {
		return 0, nil
	}

	return phase0.Slot(uint64(headSlot) - f.config.LagSlots), nil
}

// getWallclockHeadSlot returns the current head slot based on wallclock time.
func (f *FillIterator) getWallclockHeadSlot() (phase0.Slot, error) {
	metadata := f.pool.Metadata()
	if metadata == nil {
		return 0, errors.New("metadata not available")
	}

	wallclock := metadata.Wallclock()
	if wallclock == nil {
		return 0, errors.New("wallclock not available")
	}

	slot := wallclock.Slots().Current()

	return phase0.Slot(slot.Number()), nil
}

// getActivationSlot returns the slot at which the activation fork started.
func (f *FillIterator) getActivationSlot() (phase0.Slot, error) {
	// Phase0 is always active from slot 0
	if f.activationFork == spec.DataVersionPhase0 {
		return 0, nil
	}

	metadata := f.pool.Metadata()
	if metadata == nil {
		return 0, errors.New("metadata not available")
	}

	beaconSpec := metadata.Spec
	if beaconSpec == nil {
		return 0, errors.New("spec not available")
	}

	forkEpoch, err := beaconSpec.ForkEpochs.GetByName(f.activationFork.String())
	if err != nil {
		return 0, fmt.Errorf("failed to get fork epoch for %s: %w", f.activationFork.String(), err)
	}

	slotsPerEpoch := uint64(beaconSpec.SlotsPerEpoch)

	return phase0.Slot(uint64(forkEpoch.Epoch) * slotsPerEpoch), nil
}

// setCurrentSlot atomically sets the current slot and returns the new value.
func (f *FillIterator) setCurrentSlot(slot phase0.Slot) phase0.Slot {
	f.currentSlotMu.Lock()
	defer f.currentSlotMu.Unlock()

	f.currentSlot = slot

	return slot
}

// incrementCurrentSlot atomically increments the current slot.
func (f *FillIterator) incrementCurrentSlot() {
	f.currentSlotMu.Lock()
	defer f.currentSlotMu.Unlock()

	f.currentSlot++
}

// checkActivationFork checks if the slot is at or after the activation fork.
func (f *FillIterator) checkActivationFork(slot phase0.Slot) error {
	// Phase0 is always active
	if f.activationFork == spec.DataVersionPhase0 {
		return nil
	}

	activationSlot, err := f.getActivationSlot()
	if err != nil {
		return err
	}

	if slot < activationSlot {
		return fmt.Errorf("slot %d is before fork activation at slot %d", slot, activationSlot)
	}

	return nil
}

// UpdateLocation persists the current position after successful processing.
func (f *FillIterator) UpdateLocation(ctx context.Context, position *cldataIterator.Position) error {
	// Get current location from coordinator
	location, err := f.coordinator.GetHorizonLocation(ctx, f.horizonType, f.networkID)
	if err != nil {
		// Treat as new location if not found
		location = nil
	}

	// Create or update the location - only update fill_slot
	var headSlot uint64

	var fillSlot uint64

	if location != nil {
		headSlot = location.HeadSlot
		// Only update fill_slot if the new position is greater
		fillSlot = max(uint64(position.Slot), location.FillSlot)
	} else {
		// New location - initialize both
		headSlot = uint64(position.Slot)
		fillSlot = uint64(position.Slot)
	}

	newLocation := &xatu.HorizonLocation{
		NetworkId: f.networkID,
		Type:      f.horizonType,
		HeadSlot:  headSlot,
		FillSlot:  fillSlot,
	}

	if err := f.coordinator.UpsertHorizonLocation(ctx, newLocation); err != nil {
		return fmt.Errorf("failed to upsert horizon location: %w", err)
	}

	// Update metrics
	f.metrics.processedTotal.WithLabelValues(
		f.horizonType.String(),
		f.networkName,
	).Inc()
	f.metrics.positionSlot.WithLabelValues(
		f.horizonType.String(),
		f.networkName,
	).Set(float64(position.Slot))

	f.log.WithFields(logrus.Fields{
		"slot":      position.Slot,
		"head_slot": headSlot,
		"fill_slot": fillSlot,
	}).Debug("Updated horizon location (fill)")

	return nil
}

// Stop stops the FILL iterator.
func (f *FillIterator) Stop(_ context.Context) error {
	close(f.done)

	f.log.Info("FILL iterator stopped")

	return nil
}

// CurrentSlot returns the current slot position of the iterator.
func (f *FillIterator) CurrentSlot() phase0.Slot {
	f.currentSlotMu.RLock()
	defer f.currentSlotMu.RUnlock()

	return f.currentSlot
}

// HorizonType returns the horizon type this iterator is for.
func (f *FillIterator) HorizonType() xatu.HorizonType {
	return f.horizonType
}

// Config returns the iterator configuration.
func (f *FillIterator) Config() *FillIteratorConfig {
	return f.config
}

// Verify FillIterator implements the Iterator interface.
var _ cldataIterator.Iterator = (*FillIterator)(nil)
