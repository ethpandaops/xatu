package iterator

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/coordinator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BackfillIterator struct {
	log              logrus.FieldLogger
	relayMonitorType xatu.RelayMonitorType
	coordinator      coordinator.Client
	wallclock        *ethwallclock.EthereumBeaconChain
	networkName      string
	clientName       string
	relayName        string
	toSlot           phase0.Slot
	checkInterval    time.Duration
	batchSize        int
}

func NewBackfillIterator(
	log logrus.FieldLogger,
	networkName, clientName string,
	relayMonitorType xatu.RelayMonitorType,
	relayName string,
	coordinatorClient *coordinator.Client,
	wallclock *ethwallclock.EthereumBeaconChain,
	toSlot phase0.Slot,
	checkInterval time.Duration,
	batchSize int,
) *BackfillIterator {
	// Append ":backfill" suffix to relay name to create unique record
	relayNameWithSuffix := relayName + ":backfill"

	return &BackfillIterator{
		log: log.
			WithField("module", "relaymonitor/iterator/backfill").
			WithField("relay_monitor_type", relayMonitorType.String()).
			WithField("relay", relayName),
		networkName:      networkName,
		clientName:       clientName,
		relayMonitorType: relayMonitorType,
		relayName:        relayNameWithSuffix,
		coordinator:      *coordinatorClient,
		wallclock:        wallclock,
		toSlot:           toSlot,
		checkInterval:    checkInterval,
		batchSize:        batchSize,
	}
}

func (b *BackfillIterator) Next(ctx context.Context) (*phase0.Slot, error) {
	// Get current location from coordinator
	location, err := b.coordinator.GetRelayMonitorLocation(ctx, b.relayMonitorType, b.networkName, b.clientName, b.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	var backfillSlot uint64

	if location == nil {
		// Start from current slot if no location exists
		wallclockSlot := b.wallclock.Slots().Current()
		backfillSlot = wallclockSlot.Number()
	} else {
		slot, err := b.getSlot(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get slot from location")
		}

		backfillSlot = slot
	}

	// Check if we've reached the target
	if backfillSlot <= uint64(b.toSlot) {
		b.log.Debug("Backfill complete")

		return nil, nil //nolint:nilnil // nil slot indicates backfill complete
	}

	// Return the next slot to backfill (working backwards)
	nextSlot := phase0.Slot(backfillSlot - 1)

	return &nextSlot, nil
}

// NextBatch returns batch parameters for fetching multiple payloads at once.
// This is more efficient than fetching slot-by-slot for backfilling.
// Returns nil if backfill is complete or no work available.
func (b *BackfillIterator) NextBatch(ctx context.Context) (*BatchRequest, error) {
	// Get current location from coordinator
	location, err := b.coordinator.GetRelayMonitorLocation(ctx, b.relayMonitorType, b.networkName, b.clientName, b.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	var currentSlot uint64

	if location == nil {
		// Start from current slot if no location exists
		wallclockSlot := b.wallclock.Slots().Current()
		currentSlot = wallclockSlot.Number()
	} else {
		slot, err := b.getSlot(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get slot from location")
		}

		currentSlot = slot
	}

	// Check if we've reached the target
	if currentSlot <= uint64(b.toSlot) {
		b.log.Debug("Backfill complete")

		return nil, nil //nolint:nilnil // nil indicates backfill complete
	}

	// Build batch request parameters
	// Use cursor=currentSlot to fetch payloads from currentSlot going backwards
	// Explicitly request descending order to ensure we get newest slots first
	params := url.Values{
		"cursor": {fmt.Sprintf("%d", currentSlot)},
		"limit":  {fmt.Sprintf("%d", b.batchSize)},
		"order":  {"desc"},
	}

	return &BatchRequest{
		Params:      params,
		CurrentSlot: currentSlot,
		TargetSlot:  uint64(b.toSlot),
		BatchSize:   b.batchSize,
	}, nil
}

func (b *BackfillIterator) UpdateLocation(ctx context.Context, slot phase0.Slot) error {
	// Create new location with updated slot
	newLocation := b.createLocation(uint64(slot))

	b.log.WithField("slot", slot).Debug("Updating backfill location")

	return b.coordinator.UpsertRelayMonitorLocation(ctx, newLocation)
}

func (b *BackfillIterator) createLocation(slot uint64) *xatu.RelayMonitorLocation {
	location := &xatu.RelayMonitorLocation{
		MetaNetworkName: b.networkName,
		MetaClientName:  b.clientName,
		RelayName:       b.relayName, // Already includes ":backfill" suffix
		Type:            b.relayMonitorType,
	}

	// For backfill, we only use current_slot field
	marker := &xatu.RelayMonitorSlotMarker{
		CurrentSlot: slot,
	}

	switch b.relayMonitorType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		location.Data = &xatu.RelayMonitorLocation_BidTrace{
			BidTrace: &xatu.RelayMonitorLocationBidTrace{
				SlotMarker: marker,
			},
		}
	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		location.Data = &xatu.RelayMonitorLocation_PayloadDelivered{
			PayloadDelivered: &xatu.RelayMonitorLocationPayloadDelivered{
				SlotMarker: marker,
			},
		}
	}

	return location
}

func (b *BackfillIterator) getSlot(location *xatu.RelayMonitorLocation) (uint64, error) {
	if location == nil {
		return 0, errors.New("location is nil")
	}

	switch b.relayMonitorType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		data := location.GetBidTrace()
		if data == nil {
			return 0, errors.New("bid trace data is nil")
		}

		if data.SlotMarker == nil {
			// Initialize with current wallclock slot
			wallclockSlot := b.wallclock.Slots().Current()

			return wallclockSlot.Number(), nil
		}

		return data.SlotMarker.CurrentSlot, nil

	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		data := location.GetPayloadDelivered()
		if data == nil {
			return 0, errors.New("payload delivered data is nil")
		}

		if data.SlotMarker == nil {
			// Initialize with current wallclock slot
			wallclockSlot := b.wallclock.Slots().Current()

			return wallclockSlot.Number(), nil
		}

		return data.SlotMarker.CurrentSlot, nil

	default:
		return 0, fmt.Errorf("unknown relay monitor type: %s", b.relayMonitorType)
	}
}
