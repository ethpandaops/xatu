package iterator

import (
	"context"
	"fmt"
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
) *BackfillIterator {
	return &BackfillIterator{
		log: log.
			WithField("module", "relaymonitor/iterator/backfill").
			WithField("relay_monitor_type", relayMonitorType.String()).
			WithField("relay", relayName),
		networkName:      networkName,
		clientName:       clientName,
		relayMonitorType: relayMonitorType,
		relayName:        relayName,
		coordinator:      *coordinatorClient,
		wallclock:        wallclock,
		toSlot:           toSlot,
		checkInterval:    checkInterval,
	}
}

func (b *BackfillIterator) Next(ctx context.Context) (*phase0.Slot, error) {
	// Get current location from coordinator
	location, err := b.coordinator.GetRelayMonitorLocation(ctx, b.relayMonitorType, b.networkName, b.clientName, b.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	var backfillSlot int64

	if location == nil {
		// Start from current slot if no location exists
		currentSlot := b.wallclock.Slots().Current()
		backfillSlot = int64(currentSlot.Number())
	} else {
		marker, err := b.getMarker(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get marker from location")
		}

		backfillSlot = marker.BackfillSlot
	}

	// Check if we've reached the target
	if backfillSlot <= int64(b.toSlot) {
		b.log.Debug("Backfill complete")
		return nil, nil
	}

	// Return the next slot to backfill
	nextSlot := phase0.Slot(backfillSlot - 1)

	return &nextSlot, nil
}

func (b *BackfillIterator) UpdateLocation(ctx context.Context, slot phase0.Slot) error {
	// Get current location
	location, err := b.coordinator.GetRelayMonitorLocation(ctx, b.relayMonitorType, b.networkName, b.clientName, b.relayName)
	if err != nil {
		return errors.Wrap(err, "failed to get relay monitor location")
	}

	var currentSlot uint64

	if location == nil {
		// Initialize with current wallclock slot
		wallclockSlot := b.wallclock.Slots().Current()
		currentSlot = wallclockSlot.Number()
	} else {
		marker, err := b.getMarker(location)
		if err != nil {
			return errors.Wrap(err, "failed to get marker from location")
		}

		currentSlot = marker.CurrentSlot
	}

	// Create new location with updated backfill slot
	newLocation := b.createLocation(currentSlot, int64(slot))

	b.log.WithField("slot", slot).Debug("Updating backfill location")

	return b.coordinator.UpsertRelayMonitorLocation(ctx, newLocation)
}

func (b *BackfillIterator) createLocation(currentSlot uint64, backfillSlot int64) *xatu.RelayMonitorLocation {
	location := &xatu.RelayMonitorLocation{
		MetaNetworkName: b.networkName,
		MetaClientName:  b.clientName,
		RelayName:       b.relayName,
		Type:            b.relayMonitorType,
	}

	marker := &xatu.RelayMonitorSlotMarker{
		CurrentSlot:  currentSlot,
		BackfillSlot: backfillSlot,
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

func (b *BackfillIterator) getMarker(location *xatu.RelayMonitorLocation) (*xatu.RelayMonitorSlotMarker, error) {
	if location == nil {
		return nil, errors.New("location is nil")
	}

	switch b.relayMonitorType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		data := location.GetBidTrace()
		if data == nil {
			return nil, errors.New("bid trace data is nil")
		}

		if data.SlotMarker == nil {
			wallclockSlot := b.wallclock.Slots().Current()

			return &xatu.RelayMonitorSlotMarker{
				CurrentSlot:  wallclockSlot.Number(),
				BackfillSlot: int64(wallclockSlot.Number()),
			}, nil
		}

		return data.SlotMarker, nil

	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		data := location.GetPayloadDelivered()
		if data == nil {
			return nil, errors.New("payload delivered data is nil")
		}

		if data.SlotMarker == nil {
			wallclockSlot := b.wallclock.Slots().Current()

			return &xatu.RelayMonitorSlotMarker{
				CurrentSlot:  wallclockSlot.Number(),
				BackfillSlot: int64(wallclockSlot.Number()),
			}, nil
		}

		return data.SlotMarker, nil

	default:
		return nil, fmt.Errorf("unknown relay monitor type: %s", b.relayMonitorType)
	}
}
