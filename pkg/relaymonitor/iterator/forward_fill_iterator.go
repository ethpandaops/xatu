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
	// Append ":forward_fill" suffix to relay name to create unique record
	relayNameWithSuffix := relayName + ":forward_fill"

	return &ForwardFillIterator{
		log: log.
			WithField("module", "relaymonitor/iterator/forward_fill").
			WithField("relay_monitor_type", relayMonitorType.String()).
			WithField("relay", relayName),
		networkName:      networkName,
		clientName:       clientName,
		relayMonitorType: relayMonitorType,
		relayName:        relayNameWithSuffix,
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

	var currentSlot uint64

	if location == nil {
		// Start from current wallclock slot if no location exists
		currentSlot = wallclockSlot.Number()
	} else {
		slot, err := f.getSlot(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get slot from location")
		}

		currentSlot = slot
	}

	// Check if we're caught up
	if currentSlot >= wallclockSlot.Number() {
		// We're caught up, nothing to do
		return nil, nil //nolint:nilnil // nil slot indicates no work available
	}

	// Return the next slot to process (working forward)
	nextSlot := phase0.Slot(currentSlot + 1)

	return &nextSlot, nil
}

func (f *ForwardFillIterator) UpdateLocation(ctx context.Context, slot phase0.Slot) error {
	// Create new location with updated slot
	newLocation := f.createLocation(uint64(slot))

	f.log.WithField("slot", slot).Debug("Updating forward fill location")

	return f.coordinator.UpsertRelayMonitorLocation(ctx, newLocation)
}

func (f *ForwardFillIterator) createLocation(slot uint64) *xatu.RelayMonitorLocation {
	location := &xatu.RelayMonitorLocation{
		MetaNetworkName: f.networkName,
		MetaClientName:  f.clientName,
		RelayName:       f.relayName, // Already includes ":forward_fill" suffix
		Type:            f.relayMonitorType,
	}

	// For forward fill, we only use current_slot field
	marker := &xatu.RelayMonitorSlotMarker{
		CurrentSlot: slot,
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

func (f *ForwardFillIterator) getSlot(location *xatu.RelayMonitorLocation) (uint64, error) {
	if location == nil {
		return 0, errors.New("location is nil")
	}

	switch f.relayMonitorType {
	case xatu.RelayMonitorType_RELAY_MONITOR_BID_TRACE:
		data := location.GetBidTrace()
		if data == nil {
			return 0, errors.New("bid trace data is nil")
		}

		if data.SlotMarker == nil {
			// Start from current wallclock slot
			wallclockSlot := f.wallclock.Slots().Current()

			return wallclockSlot.Number(), nil
		}

		return data.SlotMarker.CurrentSlot, nil

	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		data := location.GetPayloadDelivered()
		if data == nil {
			return 0, errors.New("payload delivered data is nil")
		}

		if data.SlotMarker == nil {
			// Start from current wallclock slot
			wallclockSlot := f.wallclock.Slots().Current()

			return wallclockSlot.Number(), nil
		}

		return data.SlotMarker.CurrentSlot, nil

	default:
		return 0, fmt.Errorf("unknown relay monitor type: %s", f.relayMonitorType)
	}
}
