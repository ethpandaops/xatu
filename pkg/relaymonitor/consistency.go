package relaymonitor

import (
	"context"
	"fmt"
	"net/url"
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

// getEffectiveBatchSize returns the batch size for a relay, using per-relay override if configured.
func (r *RelayMonitor) getEffectiveBatchSize(relayClient *relay.Client) int {
	if limit := relayClient.MaxBatchLimit(); limit > 0 {
		return limit
	}

	return r.Config.Consistency.BatchSize
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
	batchSize := r.getEffectiveBatchSize(relayClient)

	// Track cursor support per event type
	supportsBidTraceCursor := relayClient.SupportsBidTraceCursor()
	supportsPayloadCursor := relayClient.SupportsPayloadCursor()

	if !supportsBidTraceCursor {
		log.Info("Relay does not support cursor for bid traces - using slot-by-slot fetching")
	}

	if !supportsPayloadCursor {
		log.Info("Relay does not support cursor for payloads - using slot-by-slot fetching")
	}

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
			batchSize,
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
				batchSize,
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
			batchSize,
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
				batchSize,
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
				supportsBidTraceCursor,
				supportsPayloadCursor,
			)
		}
	}
}

// processConsistencyWork checks iterators in priority order and processes work.
// Priority order (payload delivered first as it's "what actually happened"):
//  1. Forward fill payload delivered
//  2. Backfill payload delivered
//  3. Forward fill bid traces
//  4. Backfill bid traces
//
// Uses batch fetching when cursor is supported, otherwise falls back to slot-by-slot.
func (r *RelayMonitor) processConsistencyWork(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	forwardFillBidTrace, forwardFillPayload *iterator.ForwardFillIterator,
	backfillBidTrace, backfillPayload *iterator.BackfillIterator,
	supportsBidTraceCursor, supportsPayloadCursor bool,
) {
	network := r.Config.Ethereum.Network

	// Priority 1: Forward fill payload delivered
	if forwardFillPayload != nil {
		if r.tryProcess(
			ctx, log, relayClient, limiter, metrics,
			forwardFillPayload, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED, "forward_fill", network,
			supportsPayloadCursor,
		) {
			return
		}
	}

	// Priority 2: Backfill payload delivered
	if backfillPayload != nil {
		if r.tryProcess(
			ctx, log, relayClient, limiter, metrics,
			backfillPayload, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED, "backfill", network,
			supportsPayloadCursor,
		) {
			return
		}
	}

	// Priority 3: Forward fill bid traces
	if forwardFillBidTrace != nil {
		if r.tryProcess(
			ctx, log, relayClient, limiter, metrics,
			forwardFillBidTrace, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, "forward_fill", network,
			supportsBidTraceCursor,
		) {
			return
		}
	}

	// Priority 4: Backfill bid traces
	if backfillBidTrace != nil {
		if r.tryProcess(
			ctx, log, relayClient, limiter, metrics,
			backfillBidTrace, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE, "backfill", network,
			supportsBidTraceCursor,
		) {
			return
		}
	}
}

// tryProcess attempts to process work from an iterator.
// Uses batch fetching when cursor is supported, otherwise falls back to slot-by-slot.
// Returns true if work was processed, false if no work was available.
func (r *RelayMonitor) tryProcess(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	iter BatchIterator,
	eventType xatu.RelayMonitorType,
	process string,
	network string,
	supportsCursor bool,
) bool {
	if supportsCursor {
		return r.tryProcessBatch(ctx, log, relayClient, limiter, metrics, iter, eventType, process, network)
	}

	return r.tryProcessSlot(ctx, log, relayClient, limiter, metrics, iter, eventType, process, network)
}

// SlotIterator is an interface for iterators that can provide the next slot to process.
type SlotIterator interface {
	Next(ctx context.Context) (*phase0.Slot, error)
	UpdateLocation(ctx context.Context, slot phase0.Slot) error
}

// BatchIterator is an interface for iterators that support batch fetching.
type BatchIterator interface {
	SlotIterator
	NextBatch(ctx context.Context) (*iterator.BatchRequest, error)
}

// tryProcessBatch attempts to get a batch from an iterator and process it.
// Returns true if a batch was processed, false if no work was available.
func (r *RelayMonitor) tryProcessBatch(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	iter BatchIterator,
	eventType xatu.RelayMonitorType,
	process string,
	network string,
) bool {
	batch, err := iter.NextBatch(ctx)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{
			"event_type": eventType.String(),
			"process":    process,
		}).Error("Failed to get next batch")

		return false
	}

	if batch == nil {
		// No work available - set lag to 0
		metrics.SetLag(process, relayClient.Name(), eventType.String(), network, 0)

		return false
	}

	// Process the batch
	r.processBatch(ctx, log, relayClient, limiter, metrics, batch, eventType, process, iter, network)

	return true
}

// tryProcessSlot attempts to get the next slot from an iterator and process it.
// Used for relays that don't support cursor-based pagination.
// Returns true if a slot was processed, false if no work was available.
func (r *RelayMonitor) tryProcessSlot(
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

	// Process the single slot
	r.processSlot(ctx, log, relayClient, limiter, metrics, *slot, eventType, process, iter, network)

	return true
}

// processSlot applies rate limiting, fetches data for a single slot, and updates location on success.
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
		"mode":       "slot-by-slot",
	})

	// Update current slot metric
	metrics.SetCurrentSlot(process, relayClient.Name(), eventType.String(), network, uint64(slot))

	// Calculate and update lag metric
	r.updateLagMetric(metrics, process, relayClient.Name(), eventType, network, slot)

	log.Debug("Processing slot")

	// Apply rate limiting
	if err := limiter.Wait(ctx); err != nil {
		log.WithError(err).Debug("Rate limiter cancelled")

		return
	}

	// Build params with slot filter
	params := url.Values{
		"slot": {fmt.Sprintf("%d", slot)},
	}

	// Fetch data for this slot
	var err error

	switch eventType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		_, _, _, err = r.fetchBidTracesBatch(ctx, relayClient, params)
	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		_, _, _, err = r.fetchProposerPayloadDeliveredBatch(ctx, relayClient, params)
	}

	if err != nil {
		log.WithError(err).Error("Failed to fetch slot data")

		return // Don't update location - will retry on next tick
	}

	// Update coordinator location on success
	if err := iter.UpdateLocation(ctx, slot); err != nil {
		log.WithError(err).Error("Failed to update location")
	}
}

// processBatch applies rate limiting, fetches batch data, and updates location on success.
func (r *RelayMonitor) processBatch(
	ctx context.Context,
	log logrus.FieldLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	metrics *iterator.ConsistencyMetrics,
	batch *iterator.BatchRequest,
	eventType xatu.RelayMonitorType,
	process string,
	iter BatchIterator,
	network string,
) {
	log = log.WithFields(logrus.Fields{
		"current_slot": batch.CurrentSlot,
		"target_slot":  batch.TargetSlot,
		"event_type":   eventType.String(),
		"process":      process,
	})

	// Update current slot metric
	metrics.SetCurrentSlot(process, relayClient.Name(), eventType.String(), network, batch.CurrentSlot)

	// Calculate and update lag metric
	r.updateLagMetric(metrics, process, relayClient.Name(), eventType, network, phase0.Slot(batch.CurrentSlot))

	log.Debug("Processing batch")

	// Apply rate limiting (single consumer - no contention)
	if err := limiter.Wait(ctx); err != nil {
		log.WithError(err).Debug("Rate limiter cancelled")

		return
	}

	// Fetch batch data
	var (
		highestSlot uint64
		lowestSlot  uint64
		count       int
		err         error
	)

	switch eventType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		highestSlot, lowestSlot, count, err = r.fetchBidTracesBatch(ctx, relayClient, batch.Params)
	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		highestSlot, lowestSlot, count, err = r.fetchProposerPayloadDeliveredBatch(ctx, relayClient, batch.Params)
	}

	if err != nil {
		log.WithError(err).Error("Failed to fetch batch data")

		return // Don't update location - will retry on next tick
	}

	log.WithFields(logrus.Fields{
		"payloads_fetched": count,
		"highest_slot":     highestSlot,
		"lowest_slot":      lowestSlot,
	}).Debug("Batch fetch completed")

	// Determine the new location based on process type
	var newLocation uint64

	switch process {
	case "forward_fill":
		// For forward fill, we walk up from currentSlot in batchSize steps
		// cursor = min(currentSlot + batchSize, targetSlot)
		// If we got results, use highest slot; otherwise use cursor position
		// (which is min(currentSlot + batchSize, targetSlot))
		if count > 0 {
			newLocation = highestSlot
		} else {
			// No results - advance to the cursor position we queried
			// cursor = min(currentSlot + batchSize, targetSlot)
			//nolint:gosec // BatchSize is validated to be positive and <= 200
			cursor := batch.CurrentSlot + uint64(batch.BatchSize)
			if cursor > batch.TargetSlot {
				cursor = batch.TargetSlot
			}

			newLocation = cursor
		}
	case "backfill":
		// For backfill, we've covered down to the lowest slot in batch
		// If we got fewer than limit results, we've reached the end of available data
		switch {
		case count < batch.BatchSize && lowestSlot > batch.TargetSlot:
			// No more data available above target - we're done with this range
			newLocation = batch.TargetSlot
		case lowestSlot > 0:
			// More data may exist - update to lowest processed
			newLocation = lowestSlot
		default:
			// No results - update to target to mark complete
			newLocation = batch.TargetSlot
		}
	}

	// Update coordinator location on success
	if err := iter.UpdateLocation(ctx, phase0.Slot(newLocation)); err != nil {
		log.WithError(err).Error("Failed to update location")
	}
}

// fetchBidTracesBatch fetches bid traces using batch parameters.
// Returns highest slot, lowest slot, count of results, and any error.
func (r *RelayMonitor) fetchBidTracesBatch(
	ctx context.Context,
	client *relay.Client,
	params url.Values,
) (highestSlot, lowestSlot uint64, count int, err error) {
	requestedAt := time.Now()

	bids, err := client.GetBids(ctx, params)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get bids batch: %w", err)
	}

	responseAt := time.Now()

	if len(bids) == 0 {
		return 0, 0, 0, nil
	}

	// Track highest and lowest slots
	highestSlot = 0
	lowestSlot = ^uint64(0) // max uint64

	for _, bid := range bids {
		slot := phase0.Slot(bid.Slot.GetValue())

		if bid.Slot.GetValue() > highestSlot {
			highestSlot = bid.Slot.GetValue()
		}

		if bid.Slot.GetValue() < lowestSlot {
			lowestSlot = bid.Slot.GetValue()
		}

		// Skip if we've already seen this bid
		if r.bidCache.Has(client.Name(), slot, bid.BlockHash.GetValue()) {
			continue
		}

		r.bidCache.Set(client.Name(), slot, bid.BlockHash.GetValue())

		event, err := r.createNewDecoratedEvent(ctx, client, slot, bid, requestedAt, responseAt)
		if err != nil {
			return highestSlot, lowestSlot, len(bids), fmt.Errorf("failed to create decorated event: %w", err)
		}

		if err := r.handleNewDecoratedEvent(ctx, event); err != nil {
			r.log.WithError(err).Error("Failed to handle new decorated event")
		}
	}

	return highestSlot, lowestSlot, len(bids), nil
}

// fetchProposerPayloadDeliveredBatch fetches payload delivered using batch parameters.
// Returns highest slot, lowest slot, count of results, and any error.
func (r *RelayMonitor) fetchProposerPayloadDeliveredBatch(
	ctx context.Context,
	client *relay.Client,
	params url.Values,
) (highestSlot, lowestSlot uint64, count int, err error) {
	requestedAt := time.Now()

	payloads, err := client.GetProposerPayloadDelivered(ctx, params)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get proposer payload delivered batch: %w", err)
	}

	responseAt := time.Now()

	if len(payloads) == 0 {
		return 0, 0, 0, nil
	}

	// Track highest and lowest slots
	highestSlot = 0
	lowestSlot = ^uint64(0) // max uint64

	for _, payload := range payloads {
		slot := phase0.Slot(payload.Slot.GetValue())

		if payload.Slot.GetValue() > highestSlot {
			highestSlot = payload.Slot.GetValue()
		}

		if payload.Slot.GetValue() < lowestSlot {
			lowestSlot = payload.Slot.GetValue()
		}

		// Skip if we've already seen this payload
		if r.bidCache.Has(client.Name(), slot, payload.BlockHash.GetValue()) {
			continue
		}

		r.bidCache.Set(client.Name(), slot, payload.BlockHash.GetValue())

		event, err := r.createNewPayloadDeliveredDecoratedEvent(ctx, client, slot, payload, requestedAt, responseAt)
		if err != nil {
			return highestSlot, lowestSlot, len(payloads), fmt.Errorf("failed to create decorated event: %w", err)
		}

		if err := r.handleNewDecoratedEvent(ctx, event); err != nil {
			r.log.WithError(err).Error("Failed to handle new decorated event")
		}
	}

	return highestSlot, lowestSlot, len(payloads), nil
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
