package relaymonitor

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/iterator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

// startConsistencyProcesses starts the consistency processes for all relays if configured.
// It creates a single coordinator goroutine per relay that handles all consistency work
// (both backfill and forward fill) in priority order.
func (r *RelayMonitor) startConsistencyProcesses(ctx context.Context) error {
	if r.Config.Consistency == nil {
		r.log.Info("Consistency processes are disabled")

		return nil
	}

	if r.coordinatorClient == nil {
		r.log.Warn("Consistency processes require coordinator to be configured, skipping")

		return nil
	}

	r.log.Info("Starting consistency processes")

	metrics := iterator.NewConsistencyMetrics(namespace)

	checkInterval := r.Config.Consistency.CheckEveryDuration.Duration
	if checkInterval <= 0 {
		return fmt.Errorf("invalid checkEveryDuration: must be positive, got %v", checkInterval)
	}

	// Create one coordinator per relay (single goroutine handles all work for that relay)
	for _, relayClient := range r.relays {
		limiter := rate.NewLimiter(rate.Limit(r.Config.Consistency.RateLimitPerRelay), 1)
		go r.runConsistencyCoordinator(ctx, relayClient, limiter, metrics)
	}

	return nil
}

// runConsistencyCoordinator runs a single coordinator for all consistency work on a relay.
// It processes work in priority order: forward fill first, then backfill.
// This eliminates rate limiter contention by having a single consumer per relay.
func (r *RelayMonitor) runConsistencyCoordinator(
	ctx context.Context,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
) {
	log := r.log.WithFields(logrus.Fields{
		"relay":     relayClient.Name(),
		"component": "consistency_coordinator",
	})

	checkInterval := r.Config.Consistency.CheckEveryDuration.Duration
	network := r.Config.Ethereum.Network

	// Create all iterators (owned by this coordinator, not separate goroutines)
	var forwardFillBidTrace, forwardFillPayload *iterator.ForwardFillIterator

	var backfillBidTrace, backfillPayload *iterator.BackfillIterator

	if r.Config.Consistency.ForwardFill != nil && r.Config.Consistency.ForwardFill.Enabled {
		forwardFillBidTrace = iterator.NewForwardFillIterator(
			log,
			network,
			r.Config.Name,
			xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE,
			relayClient.Name(),
			r.coordinatorClient,
			r.ethereum.Wallclock(),
			checkInterval,
			r.Config.Consistency.ForwardFill.TrailDistance,
		)

		if r.Config.FetchProposerPayloadDelivered {
			forwardFillPayload = iterator.NewForwardFillIterator(
				log,
				network,
				r.Config.Name,
				xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED,
				relayClient.Name(),
				r.coordinatorClient,
				r.ethereum.Wallclock(),
				checkInterval,
				r.Config.Consistency.ForwardFill.TrailDistance,
			)
		}
	}

	if r.Config.Consistency.Backfill != nil && r.Config.Consistency.Backfill.Enabled {
		backfillBidTrace = iterator.NewBackfillIterator(
			log,
			network,
			r.Config.Name,
			xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE,
			relayClient.Name(),
			r.coordinatorClient,
			r.ethereum.Wallclock(),
			phase0.Slot(r.Config.Consistency.Backfill.ToSlot),
			checkInterval,
		)

		if r.Config.FetchProposerPayloadDelivered {
			backfillPayload = iterator.NewBackfillIterator(
				log,
				network,
				r.Config.Name,
				xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED,
				relayClient.Name(),
				r.coordinatorClient,
				r.ethereum.Wallclock(),
				phase0.Slot(r.Config.Consistency.Backfill.ToSlot),
				checkInterval,
			)
		}
	}

	log.Info("Starting consistency coordinator")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping consistency coordinator")

			return
		case <-ticker.C:
			// Process work in priority order
			r.processConsistencyWork(
				ctx,
				log,
				relayClient,
				limiter,
				metrics,
				forwardFillBidTrace,
				forwardFillPayload,
				backfillBidTrace,
				backfillPayload,
			)
		}
	}
}

// processConsistencyWork checks iterators in priority order and processes one slot.
// Priority order (payload delivered first as it's "what actually happened"):
//  1. Forward fill payload delivered
//  2. Backfill payload delivered
//  3. Forward fill bid traces
//  4. Backfill bid traces
func (r *RelayMonitor) processConsistencyWork(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	forwardFillBidTrace, forwardFillPayload *iterator.ForwardFillIterator,
	backfillBidTrace, backfillPayload *iterator.BackfillIterator,
) {
	network := r.Config.Ethereum.Network

	// Priority 1: Forward fill payload delivered
	if forwardFillPayload != nil {
		if r.tryProcessIterator(
			ctx, log, relayClient, limiter, metrics,
			forwardFillPayload, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED, "forward_fill", network,
		) {
			return
		}
	}

	// Priority 2: Backfill payload delivered
	if backfillPayload != nil {
		if r.tryProcessIterator(
			ctx, log, relayClient, limiter, metrics,
			backfillPayload, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED, "backfill", network,
		) {
			return
		}
	}

	// Priority 3: Forward fill bid traces
	if forwardFillBidTrace != nil {
		if r.tryProcessIterator(
			ctx, log, relayClient, limiter, metrics,
			forwardFillBidTrace, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, "forward_fill", network,
		) {
			return
		}
	}

	// Priority 4: Backfill bid traces
	if backfillBidTrace != nil {
		if r.tryProcessIterator(
			ctx, log, relayClient, limiter, metrics,
			backfillBidTrace, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, "backfill", network,
		) {
			return
		}
	}
}

// SlotIterator is an interface for iterators that can provide the next slot to process.
type SlotIterator interface {
	Next(ctx context.Context) (*phase0.Slot, error)
	UpdateLocation(ctx context.Context, slot phase0.Slot) error
}

// tryProcessIterator attempts to get the next slot from an iterator and process it.
// Returns true if a slot was processed, false if no work was available.
func (r *RelayMonitor) tryProcessIterator(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	iter SlotIterator,
	eventType xatu.RelayMonitorType,
	process string,
	network string,
) bool {
	slot, err := iter.Next(ctx)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event_type": eventType.String(),
			"process":    process,
		}).Error("Failed to get next slot")

		return false
	}

	if slot == nil {
		// No work available - set lag to 0
		metrics.SetLag(process, relayClient.Name(), eventType.String(), network, 0)

		return false
	}

	// Process the slot
	r.processSlot(ctx, log, relayClient, limiter, metrics, *slot, eventType, process, iter, network)

	return true
}

// processSlot applies rate limiting, fetches data, and updates location on success.
func (r *RelayMonitor) processSlot(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	slot phase0.Slot,
	eventType xatu.RelayMonitorType,
	process string,
	iter SlotIterator,
	network string,
) {
	log = log.WithFields(logrus.Fields{
		"slot":       slot,
		"event_type": eventType.String(),
		"process":    process,
	})

	// Update current slot metric
	metrics.SetCurrentSlot(process, relayClient.Name(), eventType.String(), network, uint64(slot))

	// Calculate and update lag metric
	r.updateLagMetric(metrics, process, relayClient.Name(), eventType, network, slot)

	log.Debug("Processing slot")

	// Apply rate limiting (single consumer - no contention)
	if err := limiter.Wait(ctx); err != nil {
		log.WithError(err).Debug("Rate limiter cancelled")

		return
	}

	// Fetch data
	var err error

	switch eventType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		err = r.fetchBidTraces(ctx, relayClient, slot)
	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		err = r.fetchProposerPayloadDelivered(ctx, relayClient, slot)
	}

	if err != nil {
		log.WithError(err).Error("Failed to fetch data")

		return // Don't update location - will retry on next tick
	}

	// Update coordinator location on success
	if err := iter.UpdateLocation(ctx, slot); err != nil {
		log.WithError(err).Error("Failed to update location")
	}
}

// updateLagMetric calculates and sets the lag metric based on process type.
func (r *RelayMonitor) updateLagMetric(
	metrics *iterator.ConsistencyMetrics,
	process string,
	relayName string,
	eventType xatu.RelayMonitorType,
	network string,
	slot phase0.Slot,
) {
	var lag int64

	switch process {
	case "forward_fill":
		// Lag is distance from current slot to max processable slot
		wallclockSlot := r.ethereum.Wallclock().Slots().Current()
		maxProcessableSlot := wallclockSlot.Number()

		if r.Config.Consistency.ForwardFill != nil &&
			wallclockSlot.Number() > r.Config.Consistency.ForwardFill.TrailDistance {
			maxProcessableSlot = wallclockSlot.Number() - r.Config.Consistency.ForwardFill.TrailDistance
		}

		lag = int64(maxProcessableSlot) - int64(slot) //nolint:gosec // slots won't overflow int64

	case "backfill":
		// Lag is distance from current slot to target slot
		if r.Config.Consistency.Backfill != nil {
			lag = int64(slot) - int64(r.Config.Consistency.Backfill.ToSlot) //nolint:gosec // slots won't overflow int64
		}
	}

	metrics.SetLag(process, relayName, eventType.String(), network, lag)
}
