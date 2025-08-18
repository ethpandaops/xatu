package relaymonitor

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// BackfillManager manages historical slot data backfilling
type BackfillManager struct {
	monitor       *RelayMonitor
	config        *BackfillConfig
	log           logrus.FieldLogger
	rateLimiters  map[string]*rate.Limiter
	cancelFunc    context.CancelFunc
	slotsPerEpoch uint64
}

// NewBackfillManager creates a new backfill manager
func NewBackfillManager(monitor *RelayMonitor) *BackfillManager {
	// Get slots per epoch from wallclock config
	// Default to 32 if not available
	slotsPerEpoch := uint64(32)
	if monitor.ethereum != nil && monitor.ethereum.Wallclock() != nil {
		// The wallclock stores slots per epoch internally
		// We'll use the standard value for now
		slotsPerEpoch = 32 // Standard Ethereum value
	}

	return &BackfillManager{
		monitor:       monitor,
		config:        monitor.Config.Backfill,
		log:           monitor.log.WithField("component", "backfill"),
		rateLimiters:  make(map[string]*rate.Limiter),
		slotsPerEpoch: slotsPerEpoch,
	}
}

// Start begins the backfill process
func (b *BackfillManager) Start(ctx context.Context) error {
	if b.config == nil || !b.config.Enabled {
		b.log.Info("Backfill is disabled")

		return nil
	}

	// Create rate limiters for each relay
	for _, relay := range b.monitor.relays {
		limiter := rate.NewLimiter(rate.Limit(b.config.RateLimit.RequestsPerSecond), 1)
		b.rateLimiters[relay.Name()] = limiter
	}

	// Calculate target slot
	targetSlot, err := b.calculateTargetSlot(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to calculate target slot")
	}

	// Get current slot
	currentSlot := b.monitor.ethereum.Wallclock().Slots().Current()

	b.log.WithFields(logrus.Fields{
		"current_slot":      currentSlot.Number(),
		"target_slot":       targetSlot,
		"slots_to_backfill": currentSlot.Number() - uint64(targetSlot),
	}).Info("Starting backfill process")

	// Create cancellable context for backfill
	backfillCtx, cancel := context.WithCancel(ctx)
	b.cancelFunc = cancel

	// Start backfill in background
	go b.runBackfill(backfillCtx, targetSlot, phase0.Slot(currentSlot.Number()))

	return nil
}

// Stop cancels the backfill process
func (b *BackfillManager) Stop() {
	if b.cancelFunc != nil {
		b.cancelFunc()
	}
}

// calculateTargetSlot determines the slot to backfill to
func (b *BackfillManager) calculateTargetSlot(_ context.Context) (phase0.Slot, error) {
	// If no target specified, go to genesis
	if b.config.To.Epoch == nil && b.config.To.Fork == nil {
		b.log.Info("No target specified, backfilling to genesis")

		return 0, nil
	}

	// If epoch specified
	if b.config.To.Epoch != nil {
		epochNum := *b.config.To.Epoch
		if epochNum == -1 {
			b.log.Info("Epoch -1 specified, backfilling to genesis")

			return 0, nil
		}

		// Calculate slot from epoch
		// Check for negative epoch
		if epochNum < 0 {
			return 0, fmt.Errorf("invalid negative epoch: %d", epochNum)
		}

		targetSlot := phase0.Slot(uint64(epochNum) * b.slotsPerEpoch)

		b.log.WithFields(logrus.Fields{
			"epoch": epochNum,
			"slot":  targetSlot,
		}).Info("Backfilling to epoch")

		return targetSlot, nil
	}

	// If fork specified
	if b.config.To.Fork != nil {
		forkName := *b.config.To.Fork

		// Map fork names to their activation epochs
		// These are hardcoded for mainnet - in a production system,
		// these should be fetched from the beacon node spec
		forkEpochs := map[string]uint64{
			"phase0":    0,
			"altair":    74240,  // Mainnet Altair fork epoch
			"bellatrix": 144896, // Mainnet Bellatrix fork epoch
			"capella":   194048, // Mainnet Capella fork epoch
			"deneb":     269568, // Mainnet Deneb fork epoch
		}

		epochNum, exists := forkEpochs[forkName]
		if !exists {
			return 0, fmt.Errorf("unknown fork name: %s", forkName)
		}

		// Calculate slot from epoch
		targetSlot := phase0.Slot(epochNum * b.slotsPerEpoch)

		b.log.WithFields(logrus.Fields{
			"fork":  forkName,
			"epoch": epochNum,
			"slot":  targetSlot,
		}).Info("Backfilling to fork")

		return targetSlot, nil
	}

	return 0, errors.New("no valid target specified")
}

// runBackfill performs the actual backfill work
func (b *BackfillManager) runBackfill(ctx context.Context, targetSlot, currentSlot phase0.Slot) {
	b.log.Info("Backfill process started")
	defer b.log.Info("Backfill process stopped")

	// Process slots in batches
	batchSize := b.config.RateLimit.SlotsPerRequest

	// Start from current slot and work backwards
	for slot := currentSlot; slot > targetSlot; {
		select {
		case <-ctx.Done():
			b.log.Info("Backfill cancelled")

			return
		default:
		}

		// Calculate batch range
		startSlot := slot

		// Calculate end slot for this batch
		// batchSize is validated to be positive in config
		var batchSizeSlots phase0.Slot
		if batchSize > 0 {
			batchSizeSlots = phase0.Slot(uint64(batchSize))
		}

		endSlot := targetSlot

		if slot > batchSizeSlots && batchSizeSlots > 0 {
			potentialEnd := slot - batchSizeSlots
			if potentialEnd > targetSlot {
				endSlot = potentialEnd
			}
		}

		// Process batch for each relay
		for _, relayClient := range b.monitor.relays {
			select {
			case <-ctx.Done():
				return
			default:
			}

			// Rate limit
			limiter := b.rateLimiters[relayClient.Name()]

			if err := limiter.Wait(ctx); err != nil {
				b.log.WithError(err).Error("Rate limiter error")

				continue
			}

			// Fetch data for the batch
			b.processBatch(ctx, relayClient, startSlot, endSlot)

			// Add delay between relays
			if b.config.RateLimit.DelayBetweenRelays.Duration > 0 {
				time.Sleep(b.config.RateLimit.DelayBetweenRelays.Duration)
			}
		}

		// Move to next batch
		slot = endSlot - 1
		if slot > currentSlot {
			// Overflow protection
			break
		}

		// Log progress periodically
		if (currentSlot-slot)%100 == 0 {
			progress := float64(currentSlot-slot) / float64(currentSlot-targetSlot) * 100
			b.log.WithFields(logrus.Fields{
				"current_batch_slot": slot,
				"target_slot":        targetSlot,
				"progress":           fmt.Sprintf("%.2f%%", progress),
			}).Info("Backfill progress")
		}
	}

	b.log.Info("Backfill completed")
}

// processBatch fetches data for a batch of slots
func (b *BackfillManager) processBatch(ctx context.Context, relayClient *relay.Client, startSlot, endSlot phase0.Slot) {
	logCtx := b.log.WithFields(logrus.Fields{
		"relay":      relayClient.Name(),
		"start_slot": startSlot,
		"end_slot":   endSlot,
	})

	// Process each slot in the batch
	for slot := startSlot; slot > endSlot; slot-- {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Fetch bid traces
		if err := b.monitor.fetchBidTraces(ctx, relayClient, slot); err != nil {
			logCtx.WithField("slot", slot).WithError(err).Debug("Failed to fetch bid traces")
		}

		// Fetch proposer payload delivered
		if b.monitor.Config.FetchProposerPayloadDelivered {
			if err := b.monitor.fetchProposerPayloadDelivered(ctx, relayClient, slot); err != nil {
				logCtx.WithField("slot", slot).WithError(err).Debug("Failed to fetch proposer payload delivered")
			}
		}
	}
}
