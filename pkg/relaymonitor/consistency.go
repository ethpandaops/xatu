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

// startConsistencyProcesses starts the consistency processes (backfill and forward fill) for all relays if configured
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

	// Create metrics instance
	metrics := iterator.NewConsistencyMetrics(namespace)

	checkInterval := r.Config.Consistency.CheckEveryDuration.Duration
	if checkInterval <= 0 {
		return fmt.Errorf("invalid checkEveryDuration: must be positive, got %v", checkInterval)
	}

	// Create a single shared rate limiter for each relay
	// Both forward fill and backfill share the same limiter, but forward fill gets priority
	// This allows backfill to use full capacity when forward fill has no work
	rateLimiters := make(map[string]*rate.Limiter)

	for _, relayClient := range r.relays {
		// Single shared rate limiter - full capacity available to both processes
		limiter := rate.NewLimiter(
			rate.Limit(r.Config.Consistency.RateLimitPerRelay),
			1, // No bursting - steady rate only to avoid relay bans
		)
		rateLimiters[relayClient.Name()] = limiter
	}

	// Create consistency iterators for each relay and event type
	for _, relayClient := range r.relays {
		limiter := rateLimiters[relayClient.Name()]

		// Start backfill process if enabled
		if r.Config.Consistency.Backfill != nil && r.Config.Consistency.Backfill.Enabled {
			// Backfill for bid traces
			go r.runBackfillIterator(ctx, relayClient, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, metrics, limiter)

			// Backfill for payload delivered if enabled
			if r.Config.FetchProposerPayloadDelivered {
				go r.runBackfillIterator(ctx, relayClient, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED, metrics, limiter)
			}
		}

		// Start forward fill process if enabled
		if r.Config.Consistency.ForwardFill != nil && r.Config.Consistency.ForwardFill.Enabled {
			// Forward fill for bid traces
			go r.runForwardFillIterator(ctx, relayClient, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, metrics, limiter)

			// Forward fill for payload delivered if enabled
			if r.Config.FetchProposerPayloadDelivered {
				go r.runForwardFillIterator(ctx, relayClient, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED, metrics, limiter)
			}
		}
	}

	return nil
}

// runBackfillIterator runs the backfill iterator for a specific relay and event type
func (r *RelayMonitor) runBackfillIterator(ctx context.Context, relayClient *relay.Client, eventType xatu.RelayMonitorType, metrics *iterator.ConsistencyMetrics, limiter *rate.Limiter) {
	log := r.log.WithFields(logrus.Fields{
		"process":    "backfill",
		"relay":      relayClient.Name(),
		"event_type": eventType.String(),
	})

	checkInterval := r.Config.Consistency.CheckEveryDuration.Duration

	// Create backfill iterator
	iter := iterator.NewBackfillIterator(
		log,
		r.Config.Ethereum.Network,
		r.Config.Name,
		eventType,
		relayClient.Name(),
		r.coordinatorClient,
		r.ethereum.Wallclock(),
		phase0.Slot(r.Config.Consistency.Backfill.ToSlot),
		checkInterval,
	)

	log.WithField("to_slot", r.Config.Consistency.Backfill.ToSlot).Info("Starting backfill iterator")

	// Main backfill loop
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping backfill iterator")

			return
		case <-ticker.C:
			// Get next slot to process
			nextSlot, err := iter.Next(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to get next backfill slot")

				continue
			}

			// Check if backfill is complete
			if nextSlot == nil {
				log.Debug("Backfill complete or no work available")

				metrics.SetLag("backfill", relayClient.Name(), eventType.String(), r.Config.Ethereum.Network, 0)

				continue
			}

			// Update metrics
			metrics.SetCurrentSlot("backfill", relayClient.Name(), eventType.String(), r.Config.Ethereum.Network, uint64(*nextSlot))
			lag := int64(*nextSlot) - int64(r.Config.Consistency.Backfill.ToSlot)
			metrics.SetLag("backfill", relayClient.Name(), eventType.String(), r.Config.Ethereum.Network, lag)

			// Process the slot
			log.WithField("slot", *nextSlot).Debug("Processing backfill slot")

			// Apply rate limiting - backfill uses WaitN with smaller timeout to yield to forward fill
			// This gives forward fill priority by allowing backfill to be interrupted
			backfillCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
			err = limiter.Wait(backfillCtx)

			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					// Rate limiter busy, yield to forward fill
					continue
				}

				log.WithError(err).Error("Rate limiter error")

				continue
			}

			// Fetch data for the slot
			switch eventType {
			case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
				err = r.fetchBidTraces(ctx, relayClient, *nextSlot)
			case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
				err = r.fetchProposerPayloadDelivered(ctx, relayClient, *nextSlot)
			}

			if err != nil {
				log.WithError(err).WithField("slot", *nextSlot).Error("Failed to fetch data")

				continue
			}

			// Update location
			if err := iter.UpdateLocation(ctx, *nextSlot); err != nil {
				log.WithError(err).Error("Failed to update location")
			}
		}
	}
}

// runForwardFillIterator runs the forward fill iterator for a specific relay and event type
func (r *RelayMonitor) runForwardFillIterator(ctx context.Context, relayClient *relay.Client, eventType xatu.RelayMonitorType, metrics *iterator.ConsistencyMetrics, limiter *rate.Limiter) {
	log := r.log.WithFields(logrus.Fields{
		"process":    "forward_fill",
		"relay":      relayClient.Name(),
		"event_type": eventType.String(),
	})

	checkInterval := r.Config.Consistency.CheckEveryDuration.Duration

	// Create forward fill iterator
	iter := iterator.NewForwardFillIterator(
		log,
		r.Config.Ethereum.Network,
		r.Config.Name,
		eventType,
		relayClient.Name(),
		r.coordinatorClient,
		r.ethereum.Wallclock(),
		checkInterval,
	)

	log.Info("Starting forward fill iterator")

	// Main forward fill loop
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping forward fill iterator")

			return
		case <-ticker.C:
			// Get next slot to process
			nextSlot, err := iter.Next(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to get next forward fill slot")

				continue
			}

			// Check if we're caught up
			if nextSlot == nil {
				log.Debug("Forward fill caught up")
				metrics.SetLag("forward_fill", relayClient.Name(), eventType.String(), r.Config.Ethereum.Network, 0)

				continue
			}

			// Update metrics
			wallclockSlot := r.ethereum.Wallclock().Slots().Current()
			metrics.SetCurrentSlot("forward_fill", relayClient.Name(), eventType.String(), r.Config.Ethereum.Network, uint64(*nextSlot))
			lag := int64(wallclockSlot.Number()) - int64(*nextSlot)
			metrics.SetLag("forward_fill", relayClient.Name(), eventType.String(), r.Config.Ethereum.Network, lag)

			// Process the slot
			log.WithField("slot", *nextSlot).Debug("Processing forward fill slot")

			// Apply rate limiting (forward fill has priority)
			err = limiter.Wait(ctx)
			if err != nil {
				log.WithError(err).Error("Rate limiter error")

				continue
			}

			// Fetch data for the slot
			switch eventType {
			case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
				err = r.fetchBidTraces(ctx, relayClient, *nextSlot)
			case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
				err = r.fetchProposerPayloadDelivered(ctx, relayClient, *nextSlot)
			}

			if err != nil {
				log.WithError(err).WithField("slot", *nextSlot).Error("Failed to fetch data")

				continue
			}

			// Update location
			if err := iter.UpdateLocation(ctx, *nextSlot); err != nil {
				log.WithError(err).Error("Failed to update location")
			}
		}
	}
}
