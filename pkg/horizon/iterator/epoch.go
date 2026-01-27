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

	cldataIterator "github.com/ethpandaops/xatu/pkg/cldata/iterator"
)

var (
	// ErrEpochIteratorClosed is returned when the epoch iterator is closed.
	ErrEpochIteratorClosed = errors.New("epoch iterator closed")
)

// EpochIteratorConfig holds configuration for the epoch iterator.
type EpochIteratorConfig struct {
	// Enabled indicates if this iterator is enabled.
	Enabled bool `yaml:"enabled" default:"true"`
	// TriggerPercent is the percentage through an epoch at which to trigger.
	// For example, 0.5 means trigger at 50% through the epoch (midway).
	// Default is 0.5 (50%).
	TriggerPercent float64 `yaml:"triggerPercent" default:"0.5"`
}

// Validate validates the configuration.
func (c *EpochIteratorConfig) Validate() error {
	if c.TriggerPercent <= 0 || c.TriggerPercent >= 1 {
		return errors.New("triggerPercent must be between 0 and 1 (exclusive)")
	}

	return nil
}

// DefaultEpochIteratorConfig returns the default epoch iterator configuration.
func DefaultEpochIteratorConfig() EpochIteratorConfig {
	return EpochIteratorConfig{
		Enabled:        true,
		TriggerPercent: 0.5,
	}
}

// EpochIterator is an iterator that fires at a configurable point within each epoch.
// It's designed for epoch-based derivers (ProposerDuty, BeaconBlob, BeaconValidators, BeaconCommittee)
// that need to fetch data for an upcoming epoch before it starts.
//
// The iterator triggers at TriggerPercent through the current epoch (default 50%) and returns
// the NEXT epoch for processing. This allows derivers to pre-fetch epoch data before it's needed.
type EpochIterator struct {
	log         logrus.FieldLogger
	pool        *ethereum.BeaconNodePool
	coordinator *coordinator.Client
	cfg         EpochIteratorConfig
	metrics     *EpochIteratorMetrics

	// horizonType is the type of deriver this iterator is for.
	horizonType xatu.HorizonType
	// networkID is the network identifier.
	networkID string
	// networkName is the human-readable network name.
	networkName string

	// activationFork is the fork at which the deriver becomes active.
	activationFork spec.DataVersion

	// lastProcessedEpoch tracks the last epoch we returned for processing.
	lastProcessedEpoch phase0.Epoch
	epochMu            sync.RWMutex
	initialized        bool

	// done signals iterator shutdown.
	done chan struct{}
}

// EpochIteratorMetrics tracks metrics for the epoch iterator.
type EpochIteratorMetrics struct {
	processedTotal   *prometheus.CounterVec
	skippedTotal     *prometheus.CounterVec
	positionEpoch    *prometheus.GaugeVec
	triggerWaitTotal *prometheus.CounterVec
}

var (
	epochIteratorMetrics     *EpochIteratorMetrics
	epochIteratorMetricsOnce sync.Once
)

// newEpochIteratorMetrics creates new metrics for the epoch iterator.
// Uses registration that doesn't panic on duplicate registration.
func newEpochIteratorMetrics(namespace string) *EpochIteratorMetrics {
	epochIteratorMetricsOnce.Do(func() {
		epochIteratorMetrics = &EpochIteratorMetrics{
			processedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "epoch_iterator",
				Name:      "processed_total",
				Help:      "Total number of epochs processed by the epoch iterator",
			}, []string{"deriver", "network"}),

			skippedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "epoch_iterator",
				Name:      "skipped_total",
				Help:      "Total number of epochs skipped",
			}, []string{"deriver", "network", "reason"}),

			positionEpoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Namespace: namespace,
				Subsystem: "epoch_iterator",
				Name:      "position_epoch",
				Help:      "Current epoch position of the epoch iterator",
			}, []string{"deriver", "network"}),

			triggerWaitTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
				Namespace: namespace,
				Subsystem: "epoch_iterator",
				Name:      "trigger_wait_total",
				Help:      "Total number of times the iterator waited for trigger point",
			}, []string{"deriver", "network"}),
		}

		prometheus.MustRegister(
			epochIteratorMetrics.processedTotal,
			epochIteratorMetrics.skippedTotal,
			epochIteratorMetrics.positionEpoch,
			epochIteratorMetrics.triggerWaitTotal,
		)
	})

	return epochIteratorMetrics
}

// NewEpochIterator creates a new epoch iterator.
func NewEpochIterator(
	log logrus.FieldLogger,
	pool *ethereum.BeaconNodePool,
	coordinatorClient *coordinator.Client,
	cfg EpochIteratorConfig,
	horizonType xatu.HorizonType,
	networkID string,
	networkName string,
) *EpochIterator {
	return &EpochIterator{
		log: log.WithFields(logrus.Fields{
			"component":    "iterator/epoch",
			"horizon_type": horizonType.String(),
		}),
		pool:        pool,
		coordinator: coordinatorClient,
		cfg:         cfg,
		horizonType: horizonType,
		networkID:   networkID,
		networkName: networkName,
		metrics:     newEpochIteratorMetrics("xatu_horizon"),
		done:        make(chan struct{}),
	}
}

// Start initializes the iterator with the activation fork version.
func (e *EpochIterator) Start(ctx context.Context, activationFork spec.DataVersion) error {
	e.activationFork = activationFork

	// Initialize last processed epoch from coordinator.
	if err := e.initializeFromCoordinator(ctx); err != nil {
		e.log.WithError(err).Warn("Failed to initialize from coordinator, starting fresh")
	}

	e.log.WithFields(logrus.Fields{
		"activation_fork": activationFork.String(),
		"network_id":      e.networkID,
		"trigger_percent": e.cfg.TriggerPercent,
		"last_epoch":      e.lastProcessedEpoch,
	}).Info("Epoch iterator started")

	return nil
}

// initializeFromCoordinator loads the last processed epoch from the coordinator.
func (e *EpochIterator) initializeFromCoordinator(ctx context.Context) error {
	location, err := e.coordinator.GetHorizonLocation(ctx, e.horizonType, e.networkID)
	if err != nil {
		return fmt.Errorf("failed to get horizon location: %w", err)
	}

	if location == nil {
		// No previous location, start from epoch 0.
		e.lastProcessedEpoch = 0
		e.initialized = false

		return nil
	}

	// For epoch-based derivers, we track the last processed epoch in head_slot field.
	// This is a bit of a misnomer but allows reuse of the existing HorizonLocation message.
	// head_slot field stores the last processed epoch number for epoch iterators.
	e.epochMu.Lock()
	e.lastProcessedEpoch = phase0.Epoch(location.HeadSlot)
	e.initialized = true
	e.epochMu.Unlock()

	return nil
}

// Next returns the next epoch to process.
// It waits until the trigger point within the current epoch (e.g., 50% through),
// then returns the NEXT epoch for processing.
func (e *EpochIterator) Next(ctx context.Context) (*cldataIterator.Position, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-e.done:
			return nil, ErrEpochIteratorClosed
		default:
			position, err := e.calculateNextPosition(ctx)
			if err != nil {
				if errors.Is(err, cldataIterator.ErrLocationUpToDate) {
					// Wait for the trigger point.
					if waitErr := e.waitForTriggerPoint(ctx); waitErr != nil {
						return nil, waitErr
					}

					continue
				}

				return nil, err
			}

			return position, nil
		}
	}
}

// calculateNextPosition determines the next epoch to process.
func (e *EpochIterator) calculateNextPosition(ctx context.Context) (*cldataIterator.Position, error) {
	metadata := e.pool.Metadata()
	if metadata == nil {
		return nil, errors.New("metadata not available")
	}

	wallclock := metadata.Wallclock()
	currentEpoch := wallclock.Epochs().Current()

	// Calculate the trigger slot within the current epoch.
	slotsPerEpoch := uint64(metadata.Spec.SlotsPerEpoch)
	triggerSlotOffset := uint64(float64(slotsPerEpoch) * e.cfg.TriggerPercent)
	epochStartSlot := currentEpoch.Number() * slotsPerEpoch
	triggerSlot := epochStartSlot + triggerSlotOffset

	// Get the current slot.
	currentSlot := wallclock.Slots().Current()

	// If we haven't reached the trigger point yet, we're up to date.
	if currentSlot.Number() < triggerSlot {
		return nil, cldataIterator.ErrLocationUpToDate
	}

	// The epoch to process is the NEXT epoch (current + 1).
	nextEpoch := phase0.Epoch(currentEpoch.Number() + 1)

	// Check if we already processed this epoch.
	e.epochMu.RLock()
	lastProcessed := e.lastProcessedEpoch
	initialized := e.initialized
	e.epochMu.RUnlock()

	if initialized && nextEpoch <= lastProcessed {
		// Already processed, wait for next epoch.
		return nil, cldataIterator.ErrLocationUpToDate
	}

	// Check if the activation fork is active for this epoch.
	if err := e.checkActivationFork(nextEpoch); err != nil {
		e.metrics.skippedTotal.WithLabelValues(
			e.horizonType.String(),
			e.networkName,
			"pre_activation",
		).Inc()

		e.log.WithFields(logrus.Fields{
			"epoch":  nextEpoch,
			"reason": err.Error(),
		}).Trace("Skipping epoch due to activation fork")

		// Mark as processed to move forward.
		e.epochMu.Lock()
		e.lastProcessedEpoch = nextEpoch
		e.initialized = true
		e.epochMu.Unlock()

		return nil, cldataIterator.ErrLocationUpToDate
	}

	// Create position for the epoch.
	position := &cldataIterator.Position{
		Slot:            phase0.Slot(uint64(nextEpoch) * slotsPerEpoch), // First slot of the epoch.
		Epoch:           nextEpoch,
		Direction:       cldataIterator.DirectionForward,
		LookAheadEpochs: e.calculateLookAhead(nextEpoch),
	}

	e.log.WithFields(logrus.Fields{
		"epoch":         nextEpoch,
		"current_epoch": currentEpoch.Number(),
		"current_slot":  currentSlot.Number(),
	}).Debug("Returning next epoch for processing")

	return position, nil
}

// calculateLookAhead returns the epochs to look ahead for pre-fetching.
func (e *EpochIterator) calculateLookAhead(currentEpoch phase0.Epoch) []phase0.Epoch {
	// Look ahead by 1 epoch for pre-fetching.
	return []phase0.Epoch{currentEpoch + 1}
}

// checkActivationFork checks if the epoch is after the activation fork.
func (e *EpochIterator) checkActivationFork(epoch phase0.Epoch) error {
	// Phase0 is always active.
	if e.activationFork == spec.DataVersionPhase0 {
		return nil
	}

	metadata := e.pool.Metadata()
	if metadata == nil {
		return errors.New("metadata not available")
	}

	beaconSpec := metadata.Spec
	if beaconSpec == nil {
		return errors.New("spec not available")
	}

	forkEpoch, err := beaconSpec.ForkEpochs.GetByName(e.activationFork.String())
	if err != nil {
		return fmt.Errorf("failed to get fork epoch for %s: %w", e.activationFork.String(), err)
	}

	if epoch < forkEpoch.Epoch {
		return fmt.Errorf("epoch %d is before fork activation at epoch %d", epoch, forkEpoch.Epoch)
	}

	return nil
}

// waitForTriggerPoint waits until the trigger point within the current epoch.
func (e *EpochIterator) waitForTriggerPoint(ctx context.Context) error {
	metadata := e.pool.Metadata()
	if metadata == nil {
		return errors.New("metadata not available")
	}

	wallclock := metadata.Wallclock()
	currentEpoch := wallclock.Epochs().Current()

	// Calculate the trigger time.
	slotsPerEpoch := uint64(metadata.Spec.SlotsPerEpoch)
	triggerSlotOffset := uint64(float64(slotsPerEpoch) * e.cfg.TriggerPercent)
	epochStartSlot := currentEpoch.Number() * slotsPerEpoch
	triggerSlot := epochStartSlot + triggerSlotOffset

	// Get the trigger slot's start time.
	triggerSlotInfo := wallclock.Slots().FromNumber(triggerSlot)
	triggerTime := triggerSlotInfo.TimeWindow().Start()

	// If we're past the trigger time but haven't processed, check next epoch.
	now := time.Now()
	if now.After(triggerTime) {
		// Check if we need to wait for next epoch.
		e.epochMu.RLock()
		lastProcessed := e.lastProcessedEpoch
		initialized := e.initialized
		e.epochMu.RUnlock()

		nextEpoch := phase0.Epoch(currentEpoch.Number() + 1)

		if initialized && nextEpoch <= lastProcessed {
			// We've processed this epoch, wait for next epoch's trigger point.
			nextEpochStart := (currentEpoch.Number() + 1) * slotsPerEpoch
			nextTriggerSlot := nextEpochStart + triggerSlotOffset
			nextTriggerSlotInfo := wallclock.Slots().FromNumber(nextTriggerSlot)
			triggerTime = nextTriggerSlotInfo.TimeWindow().Start()
		}
	}

	waitDuration := time.Until(triggerTime)
	if waitDuration <= 0 {
		// Already past trigger time, no need to wait.
		return nil
	}

	e.metrics.triggerWaitTotal.WithLabelValues(
		e.horizonType.String(),
		e.networkName,
	).Inc()

	e.log.WithFields(logrus.Fields{
		"wait_duration": waitDuration.String(),
		"trigger_time":  triggerTime,
		"trigger_slot":  triggerSlot,
	}).Debug("Waiting for epoch trigger point")

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-e.done:
		return ErrEpochIteratorClosed
	case <-time.After(waitDuration):
		return nil
	}
}

// UpdateLocation persists the current position after successful processing.
func (e *EpochIterator) UpdateLocation(ctx context.Context, position *cldataIterator.Position) error {
	// For epoch iterators, we store the processed epoch in the head_slot field.
	// This reuses the existing HorizonLocation structure.
	newLocation := &xatu.HorizonLocation{
		NetworkId: e.networkID,
		Type:      e.horizonType,
		HeadSlot:  uint64(position.Epoch), // Store epoch as "head_slot"
		FillSlot:  0,                      // Not used for epoch iterators.
	}

	if err := e.coordinator.UpsertHorizonLocation(ctx, newLocation); err != nil {
		return fmt.Errorf("failed to upsert horizon location: %w", err)
	}

	// Update local tracking.
	e.epochMu.Lock()
	e.lastProcessedEpoch = position.Epoch
	e.initialized = true
	e.epochMu.Unlock()

	// Update metrics.
	e.metrics.processedTotal.WithLabelValues(
		e.horizonType.String(),
		e.networkName,
	).Inc()
	e.metrics.positionEpoch.WithLabelValues(
		e.horizonType.String(),
		e.networkName,
	).Set(float64(position.Epoch))

	e.log.WithFields(logrus.Fields{
		"epoch": position.Epoch,
	}).Debug("Updated epoch location")

	return nil
}

// Stop stops the epoch iterator.
func (e *EpochIterator) Stop(_ context.Context) error {
	close(e.done)

	e.log.Info("Epoch iterator stopped")

	return nil
}

// HorizonType returns the horizon type this iterator is for.
func (e *EpochIterator) HorizonType() xatu.HorizonType {
	return e.horizonType
}

// LastProcessedEpoch returns the last processed epoch.
func (e *EpochIterator) LastProcessedEpoch() phase0.Epoch {
	e.epochMu.RLock()
	defer e.epochMu.RUnlock()

	return e.lastProcessedEpoch
}

// Verify EpochIterator implements the Iterator interface.
var _ cldataIterator.Iterator = (*EpochIterator)(nil)
