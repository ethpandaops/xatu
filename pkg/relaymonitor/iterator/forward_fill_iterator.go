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

// DefaultBatchSize is the default number of payloads to fetch per batch.
const DefaultBatchSize = 200

// BatchRequest contains parameters for a batch fetch request.
type BatchRequest struct {
	// Params are the URL query parameters to pass to the relay API.
	Params url.Values
	// CurrentSlot is the current position of the iterator.
	CurrentSlot uint64
	// TargetSlot is the slot we're trying to reach.
	TargetSlot uint64
	// BatchSize is the number of payloads requested in this batch.
	BatchSize int
}

type ForwardFillIterator struct {
	log              logrus.FieldLogger
	relayMonitorType xatu.RelayMonitorType
	coordinator      coordinator.Client
	wallclock        *ethwallclock.EthereumBeaconChain
	networkName      string
	clientName       string
	relayName        string
	checkInterval    time.Duration
	trailDistance    uint64
	batchSize        int
}

func NewForwardFillIterator(
	log logrus.FieldLogger,
	networkName, clientName string,
	relayMonitorType xatu.RelayMonitorType,
	relayName string,
	coordinatorClient *coordinator.Client,
	wallclock *ethwallclock.EthereumBeaconChain,
	checkInterval time.Duration,
	trailDistance uint64,
	batchSize int,
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
		trailDistance:    trailDistance,
		batchSize:        batchSize,
	}
}

func (f *ForwardFillIterator) Next(ctx context.Context) (*phase0.Slot, error) {
	wallclockSlot := f.wallclock.Slots().Current()

	// Calculate the maximum slot we should process (head slot - trail distance)
	var maxSlot uint64
	if wallclockSlot.Number() > f.trailDistance {
		maxSlot = wallclockSlot.Number() - f.trailDistance
	} else {
		// If wallclock is less than trail distance, we don't process anything
		return nil, nil //nolint:nilnil // nil slot indicates no work available
	}

	// Get current location from coordinator
	location, err := f.coordinator.GetRelayMonitorLocation(ctx, f.relayMonitorType, f.networkName, f.clientName, f.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	var currentSlot uint64

	if location == nil {
		// Start processing from the beginning of the trail window
		// We subtract 1 to ensure we process at least maxSlot on the first iteration
		if maxSlot > 0 {
			currentSlot = maxSlot - 1
		} else {
			currentSlot = 0
		}
	} else {
		slot, err := f.getSlot(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get slot from location")
		}

		currentSlot = slot
	}

	// Check if we're caught up (considering trail distance)
	if currentSlot >= maxSlot {
		// We're caught up, nothing to do
		return nil, nil //nolint:nilnil // nil slot indicates no work available
	}

	// Return the next slot to process (working forward)
	nextSlot := phase0.Slot(currentSlot + 1)

	return &nextSlot, nil
}

// NextBatch returns batch parameters for fetching multiple payloads at once.
// This is more efficient than fetching slot-by-slot for catching up.
// Returns nil if already caught up or no work available.
func (f *ForwardFillIterator) NextBatch(ctx context.Context) (*BatchRequest, error) {
	wallclockSlot := f.wallclock.Slots().Current()

	// Calculate the maximum slot we should process (head slot - trail distance)
	var targetSlot uint64
	if wallclockSlot.Number() > f.trailDistance {
		targetSlot = wallclockSlot.Number() - f.trailDistance
	} else {
		// If wallclock is less than trail distance, we don't process anything
		return nil, nil //nolint:nilnil // nil indicates no work available
	}

	// Get current location from coordinator
	location, err := f.coordinator.GetRelayMonitorLocation(ctx, f.relayMonitorType, f.networkName, f.clientName, f.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	var currentSlot uint64

	if location == nil {
		// Start processing from the beginning of the trail window
		if targetSlot > 0 {
			currentSlot = targetSlot - 1
		} else {
			currentSlot = 0
		}
	} else {
		slot, err := f.getSlot(location)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get slot from location")
		}

		currentSlot = slot
	}

	// Check if we're caught up (considering trail distance)
	if currentSlot >= targetSlot {
		// We're caught up, nothing to do
		return nil, nil //nolint:nilnil // nil indicates no work available
	}

	// Build batch request parameters
	// Use cursor=targetSlot to fetch payloads from targetSlot going backwards
	// Explicitly request descending order to ensure we get newest slots first
	params := url.Values{
		"cursor": {fmt.Sprintf("%d", targetSlot)},
		"limit":  {fmt.Sprintf("%d", f.batchSize)},
		"order":  {"desc"},
	}

	return &BatchRequest{
		Params:      params,
		CurrentSlot: currentSlot,
		TargetSlot:  targetSlot,
		BatchSize:   f.batchSize,
	}, nil
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
			// Start from current wallclock slot minus trail distance
			wallclockSlot := f.wallclock.Slots().Current()
			if wallclockSlot.Number() > f.trailDistance {
				return wallclockSlot.Number() - f.trailDistance, nil
			}

			return 0, nil
		}

		return data.SlotMarker.CurrentSlot, nil

	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		data := location.GetPayloadDelivered()
		if data == nil {
			return 0, errors.New("payload delivered data is nil")
		}

		if data.SlotMarker == nil {
			// Start from current wallclock slot minus trail distance
			wallclockSlot := f.wallclock.Slots().Current()
			if wallclockSlot.Number() > f.trailDistance {
				return wallclockSlot.Number() - f.trailDistance, nil
			}

			return 0, nil
		}

		return data.SlotMarker.CurrentSlot, nil

	default:
		return 0, fmt.Errorf("unknown relay monitor type: %s", f.relayMonitorType)
	}
}
