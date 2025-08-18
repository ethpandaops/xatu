package relaymonitor

import (
	"context"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/iterator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/relay"
	"github.com/sirupsen/logrus"
)

// startBackfilling starts the backfill process for all relays if configured
func (r *RelayMonitor) startBackfilling(ctx context.Context) error {
	if r.Config.Backfill == nil || !r.Config.Backfill.Enabled {
		r.log.Info("Backfilling is disabled")
		return nil
	}

	if r.coordinatorClient == nil {
		r.log.Warn("Backfilling requires coordinator to be configured, skipping")
		return nil
	}

	r.log.Info("Starting backfill process")

	// Create backfill iterators for each relay and event type
	for _, relayClient := range r.relays {
		// Start backfill for bid traces
		go r.runBackfillIterator(ctx, relayClient, xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE)

		// Start backfill for payload delivered if enabled
		if r.Config.FetchProposerPayloadDelivered {
			go r.runBackfillIterator(ctx, relayClient, xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED)
		}
	}

	return nil
}

// runBackfillIterator runs the backfill iterator for a specific relay and event type
func (r *RelayMonitor) runBackfillIterator(ctx context.Context, relayClient *relay.Client, eventType xatu.RelayMonitorType) {
	log := r.log.WithFields(logrus.Fields{
		"relay":      relayClient.Name(),
		"event_type": eventType.String(),
	})

	// Create iterator config
	iterConfig := &iterator.BackfillingSlotConfig{
		Enabled:            r.Config.Backfill.Enabled,
		MinimumSlot:        phase0.Slot(r.Config.Backfill.MinimumSlot),
		CheckEveryDuration: r.Config.Backfill.CheckEveryDuration.Duration,
	}

	// Create the backfilling iterator with the instance name
	iter := iterator.NewBackfillingSlot(
		log,
		r.Config.Ethereum.Network, // network name
		r.Config.Name,             // client instance name
		eventType,
		relayClient.Name(),
		r.coordinatorClient,
		r.ethereum.Wallclock(),
		r.ethereum,
		iterConfig,
	)

	// Start the iterator
	if err := iter.Start(ctx); err != nil {
		log.WithError(err).Error("Failed to start backfill iterator")
		return
	}

	// Main backfill loop
	ticker := time.NewTicker(iterConfig.CheckEveryDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("Stopping backfill iterator")
			return
		case <-ticker.C:
			// Get next slot to process
			next, err := iter.Next(ctx)
			if err != nil {
				log.WithError(err).Error("Failed to get next slot")
				continue
			}

			// Process the slot
			log.WithFields(logrus.Fields{
				"slot":      next.Next,
				"direction": next.Direction,
			}).Debug("Processing backfill slot")

			// Fetch data for the slot
			switch eventType {
			case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
				err = r.fetchBidTraces(ctx, relayClient, next.Next)
			case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
				err = r.fetchProposerPayloadDelivered(ctx, relayClient, next.Next)
			}

			if err != nil {
				log.WithError(err).WithField("slot", next.Next).Error("Failed to fetch data")
				continue
			}

			// Update location
			if err := iter.UpdateLocation(ctx, next.Next, next.Direction); err != nil {
				log.WithError(err).Error("Failed to update location")
			}
		}
	}
}
