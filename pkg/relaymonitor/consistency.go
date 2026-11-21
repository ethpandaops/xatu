package relaymonitor

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/iterator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
)

// startConsistencyProcesses starts the consistency processes for all relays if configured.
// It creates a single coordinator goroutine per relay that handles all consistency work
// (both backfill and forward fill) in priority order.
func (r *RelayMonitor) startConsistencyProcesses(ctx context.Context) error {
	if r.Config.Consistency == nil {
		r.log.WithContext(ctx).Info("Consistency processes are disabled")

		return nil
	}

	if r.coordinatorClient == nil {
		r.log.WithContext(ctx).Warn("Consistency processes require coordinator to be configured, skipping")

		return nil
	}

	r.log.WithContext(ctx).Info("Starting consistency processes")

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
		log.WithContext(ctx).Info("Relay does not support cursor for bid traces - using slot-by-slot fetching")
	}

	if !supportsPayloadCursor {
		log.WithContext(ctx).Info("Relay does not support cursor for payloads - using slot-by-slot fetching")
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

	log.WithContext(ctx).Info("Starting consistency coordinator")

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.WithContext(ctx).Info("Stopping consistency coordinator")

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
	log observability.ContextualLogger, relayClient *relay.Client,
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
	log observability.ContextualLogger, relayClient *relay.Client,
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
	log observability.ContextualLogger, relayClient *relay.Client,
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
		}).WithContext(ctx).Error("Failed to get next batch")

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
	log observability.ContextualLogger, relayClient *relay.Client,
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
		}).WithContext(ctx).Error("Failed to get next slot")

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
	log observability.ContextualLogger, relayClient *relay.Client,
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

	log.WithContext(ctx).Debug("Processing slot")

	// Apply rate limiting
	if err := limiter.Wait(ctx); err != nil {
		log.WithError(err).WithContext(ctx).Debug("Rate limiter cancelled")

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
		log.WithError(err).WithContext(ctx).Error("Failed to fetch slot data")

		return // Don't update location - will retry on next tick
	}

	// Update coordinator location on success
	if err := iter.UpdateLocation(ctx, slot); err != nil {
		log.WithError(err).WithContext(ctx).Error("Failed to update location")
	}
}

// processBatch applies rate limiting, fetches batch data, and updates location on success.
func (r *RelayMonitor) processBatch(
	ctx context.Context,
	log observability.ContextualLogger, relayClient *relay.Client,
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

	log.WithContext(ctx).Debug("Processing batch")

	var newLocation uint64

	switch process {
	case "forward_fill":
		// A single limit-capped fetch can return only the highest slots in
		// the requested range on a dense relay, leaving lower slots in the
		// window unfetched. fetchForwardFillWindow pages down through the
		// window until it is fully covered, or gives up and leaves the
		// cursor unchanged so the same window is retried next tick, rather
		// than advancing past slots that were never actually fetched.
		var (
			progressed bool
			err        error
		)

		newLocation, progressed, err = r.fetchForwardFillWindow(ctx, log, relayClient, limiter, eventType, batch)
		if err != nil {
			log.WithError(err).WithContext(ctx).Error("Failed to fetch forward-fill window")

			return
		}

		if !progressed {
			// No safe progress was made this tick; retry the same window
			// next tick instead of persisting a no-op location update.
			return
		}
	case "backfill":
		// Apply rate limiting (single consumer - no contention)
		if err := limiter.Wait(ctx); err != nil {
			log.WithError(err).WithContext(ctx).Debug("Rate limiter cancelled")

			return
		}

		var (
			lowestSlot uint64
			count      int
			err        error
		)

		switch eventType {
		case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
			_, lowestSlot, count, err = r.fetchBidTracesBatch(ctx, relayClient, batch.Params)
		case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
			_, lowestSlot, count, err = r.fetchProposerPayloadDeliveredBatch(ctx, relayClient, batch.Params)
		}

		if err != nil {
			log.WithError(err).WithContext(ctx).Error("Failed to fetch batch data")

			return // Don't update location - will retry on next tick
		}

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
		log.WithError(err).WithContext(ctx).Error("Failed to update location")
	}
}

// maxForwardFillPagesPerTick bounds how many additional fetches a single
// forward-fill tick will issue when one page's response doesn't reach down
// to the current cursor. This keeps a tick's work bounded while still
// making real progress on dense relays instead of silently skipping the
// unfetched range.
const maxForwardFillPagesPerTick = 5

// fetchForwardFillWindow fetches the window described by batch, paging down
// through it as needed so that the returned location is never advanced past
// a slot that wasn't actually fetched. progressed is false when the window
// could not be fully covered within the page budget, in which case the
// caller should leave the location unchanged and retry next tick.
func (r *RelayMonitor) fetchForwardFillWindow(
	ctx context.Context,
	log observability.ContextualLogger,
	relayClient *relay.Client,
	limiter *rate.Limiter,
	eventType xatu.RelayMonitorType,
	batch *iterator.BatchRequest,
) (newLocation uint64, progressed bool, err error) {
	originalCursor, err := strconv.ParseUint(batch.Params.Get("cursor"), 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("failed to parse batch cursor: %w", err)
	}

	pageCursor := originalCursor
	floor := batch.CurrentSlot + 1
	covered := false
	totalCount := 0

	for page := 0; page < maxForwardFillPagesPerTick; page++ {
		if err := limiter.Wait(ctx); err != nil {
			return 0, false, err
		}

		params := url.Values{
			"cursor": {strconv.FormatUint(pageCursor, 10)},
			"limit":  {strconv.Itoa(batch.BatchSize)},
			"order":  {"desc"},
		}

		var (
			lowestSlot uint64
			count      int
			ferr       error
		)

		switch eventType {
		case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
			_, lowestSlot, count, ferr = r.fetchBidTracesBatch(ctx, relayClient, params)
		case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
			_, lowestSlot, count, ferr = r.fetchProposerPayloadDeliveredBatch(ctx, relayClient, params)
		}

		if ferr != nil {
			return 0, false, ferr
		}

		totalCount += count

		if count == 0 || count < batch.BatchSize || lowestSlot <= floor {
			// Either there's nothing left at or below pageCursor, or this
			// page wasn't truncated, or it reached our floor directly: in
			// every case the window down to floor is now fully covered.
			covered = true

			break
		}

		// Truncated and still above the floor: page further down.
		pageCursor = lowestSlot - 1
	}

	log.WithFields(logrus.Fields{
		"payloads_fetched": totalCount,
		"covered":          covered,
	}).WithContext(ctx).Debug("Forward-fill window fetch completed")

	if !covered {
		log.WithFields(logrus.Fields{
			"cursor": originalCursor,
			"floor":  floor,
		}).WithContext(ctx).Warn("Forward-fill window too dense to cover within the page budget, retrying next tick")

		return 0, false, nil
	}

	return originalCursor, true, nil
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
			r.log.WithError(err).WithContext(ctx).Error("Failed to handle new decorated event")
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
			r.log.WithError(err).WithContext(ctx).Error("Failed to handle new decorated event")
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
