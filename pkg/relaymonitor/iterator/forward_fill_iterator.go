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

type ForwardFillIterator struct {
	log              logrus.FieldLogger
	relayMonitorType xatu.RelayMonitorType
	coordinator      coordinator.Client
	wallclock        *ethwallclock.EthereumBeaconChain
	networkName      string
	clientName       string
	relayName        string
	checkInterval    time.Duration
}

func NewForwardFillIterator(
	log logrus.FieldLogger,
	networkName, clientName string,
	relayMonitorType xatu.RelayMonitorType,
	relayName string,
	coordinatorClient *coordinator.Client,
	wallclock *ethwallclock.EthereumBeaconChain,
	checkInterval time.Duration,
) *ForwardFillIterator {
	return &ForwardFillIterator{
		log: log.
			WithField("module", "relaymonitor/iterator/forward_fill").
			WithField("relay_monitor_type", relayMonitorType.String()).
			WithField("relay", relayName),
		networkName:      networkName,
		clientName:       clientName,
		relayMonitorType: relayMonitorType,
		relayName:        relayName,
		coordinator:      *coordinatorClient,
		wallclock:        wallclock,
		checkInterval:    checkInterval,
	}
}

func (f *ForwardFillIterator) Next(ctx context.Context) (*phase0.Slot, error) {
	wallclockSlot := f.wallclock.Slots().Current()

	// Get current location from coordinator
	location, err := f.coordinator.GetRelayMonitorLocation(ctx, f.relayMonitorType, f.networkName, f.clientName, f.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	if location == nil {
		// Start from current slot if no location exists
		slot := phase0.Slot(wallclockSlot.Number())

		return &slot, nil
	}

	marker, err := f.getMarker(location)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get marker from location")
	}

	// Check if we're caught up
	if marker.CurrentSlot >= wallclockSlot.Number() {
		// We're caught up, nothing to do
		return nil, nil
	}

	// Return the next slot to process
	nextSlot := phase0.Slot(marker.CurrentSlot + 1)

	return &nextSlot, nil
}

func (f *ForwardFillIterator) UpdateLocation(ctx context.Context, slot phase0.Slot) error {
	// Get current location
	location, err := f.coordinator.GetRelayMonitorLocation(ctx, f.relayMonitorType, f.networkName, f.clientName, f.relayName)
	if err != nil {
		return errors.Wrap(err, "failed to get relay monitor location")
	}

	var backfillSlot int64
	if location == nil {
		// Initialize with current slot
		backfillSlot = int64(slot)
	} else {
		marker, err := f.getMarker(location)
		if err != nil {
			return errors.Wrap(err, "failed to get marker from location")
		}

		backfillSlot = marker.BackfillSlot
	}

	// Create new location with updated current slot
	newLocation := f.createLocation(uint64(slot), backfillSlot)

	f.log.WithField("slot", slot).Debug("Updating forward fill location")

	return f.coordinator.UpsertRelayMonitorLocation(ctx, newLocation)
}

func (f *ForwardFillIterator) createLocation(currentSlot uint64, backfillSlot int64) *xatu.RelayMonitorLocation {
	location := &xatu.RelayMonitorLocation{
		MetaNetworkName: f.networkName,
		MetaClientName:  f.clientName,
		RelayName:       f.relayName,
		Type:            f.relayMonitorType,
	}

	marker := &xatu.RelayMonitorSlotMarker{
		CurrentSlot:  currentSlot,
		BackfillSlot: backfillSlot,
	}

	switch f.relayMonitorType {
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

func (f *ForwardFillIterator) getMarker(location *xatu.RelayMonitorLocation) (*xatu.RelayMonitorSlotMarker, error) {
	if location == nil {
		return nil, errors.New("location is nil")
	}

	switch f.relayMonitorType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		data := location.GetBidTrace()
		if data == nil {
			return nil, errors.New("bid trace data is nil")
		}

		if data.SlotMarker == nil {
			wallclockSlot := f.wallclock.Slots().Current()

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
			wallclockSlot := f.wallclock.Slots().Current()

			return &xatu.RelayMonitorSlotMarker{
				CurrentSlot:  wallclockSlot.Number(),
				BackfillSlot: int64(wallclockSlot.Number()),
			}, nil
		}

		return data.SlotMarker, nil

	default:
		return nil, fmt.Errorf("unknown relay monitor type: %s", f.relayMonitorType)
	}
}
