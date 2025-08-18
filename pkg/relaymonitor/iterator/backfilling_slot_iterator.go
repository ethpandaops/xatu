package iterator

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/coordinator"
	"github.com/ethpandaops/xatu/pkg/relaymonitor/ethereum"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type BackfillingSlot struct {
	log              logrus.FieldLogger
	relayMonitorType xatu.RelayMonitorType
	coordinator      coordinator.Client
	wallclock        *ethwallclock.EthereumBeaconChain
	networkName      string
	clientName       string
	relayName        string
	beaconNode       *ethereum.BeaconNode
	config           *BackfillingSlotConfig
	metrics          *BackfillingSlotMetrics
}

type BackfillingSlotDirection string

const (
	BackfillingSlotDirectionBackfill BackfillingSlotDirection = "backfill"
	BackfillingSlotDirectionHead     BackfillingSlotDirection = "head"
)

type BackfillingSlotNextResponse struct {
	Next      phase0.Slot
	Direction BackfillingSlotDirection
}

type BackfillingSlotConfig struct {
	Enabled            bool
	MinimumSlot        phase0.Slot
	CheckEveryDuration time.Duration
}

type BackfillingSlotMetrics struct {
	// Add metrics fields as needed
}

func NewBackfillingSlot(
	log logrus.FieldLogger,
	networkName, clientName string,
	relayMonitorType xatu.RelayMonitorType,
	relayName string,
	coordinatorClient *coordinator.Client,
	wallclock *ethwallclock.EthereumBeaconChain,
	beacon *ethereum.BeaconNode,
	config *BackfillingSlotConfig,
) *BackfillingSlot {
	return &BackfillingSlot{
		log: log.
			WithField("module", "relaymonitor/iterator/backfilling_slot_iterator").
			WithField("relay_monitor_type", relayMonitorType.String()).
			WithField("relay", relayName),
		networkName:      networkName,
		clientName:       clientName,
		relayMonitorType: relayMonitorType,
		relayName:        relayName,
		coordinator:      *coordinatorClient,
		wallclock:        wallclock,
		beaconNode:       beacon,
		config:           config,
		metrics:          &BackfillingSlotMetrics{},
	}
}

func (b *BackfillingSlot) Start(ctx context.Context) error {
	// Check the backfill slot is ok
	if b.config.Enabled {
		b.log.WithFields(logrus.Fields{
			"backfill_target_slot": b.config.MinimumSlot,
		}).Info("Backfilling is enabled")
	} else {
		b.log.Info("Backfilling is disabled")
	}

	return nil
}

func (b *BackfillingSlot) UpdateLocation(ctx context.Context, slot phase0.Slot, direction BackfillingSlotDirection) error {
	location, err := b.coordinator.GetRelayMonitorLocation(ctx, b.relayMonitorType, b.networkName, b.clientName, b.relayName)
	if err != nil {
		return errors.Wrap(err, "failed to get relay monitor location")
	}

	if location == nil {
		location = b.createLocationFromSlotNumber(slot, slot)
	}

	marker, err := b.GetMarker(location)
	if err != nil {
		return errors.Wrap(err, "failed to get marker from location")
	}

	switch direction {
	case BackfillingSlotDirectionHead:
		marker.CurrentSlot = uint64(slot)
	case BackfillingSlotDirectionBackfill:
		marker.BackfillSlot = int64(slot)
	default:
		return errors.Errorf("unknown direction (%s) when updating relay monitor location", direction)
	}

	newLocation := b.createLocationFromSlotNumber(phase0.Slot(marker.CurrentSlot), phase0.Slot(marker.BackfillSlot))

	b.log.WithFields(logrus.Fields{
		"direction": direction,
		"slot":      slot,
	}).Debug("Updating relay monitor location")

	err = b.coordinator.UpsertRelayMonitorLocation(ctx, newLocation)
	if err != nil {
		return errors.Wrap(err, "failed to update relay monitor location")
	}

	b.log.WithFields(logrus.Fields{
		"direction": direction,
		"slot":      slot,
	}).Debug("Updated relay monitor location")

	return nil
}

func (b *BackfillingSlot) Next(ctx context.Context) (*BackfillingSlotNextResponse, error) {
	// Get the current finalized checkpoint from the beacon node
	checkpoint, err := b.fetchLatestCheckpoint(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch latest checkpoint")
	}

	if checkpoint == nil {
		return nil, errors.New("checkpoint is nil")
	}

	// Calculate the finalized slot from the checkpoint epoch
	// Standard Ethereum has 32 slots per epoch
	slotsPerEpoch := uint64(32)
	finalizedSlot := phase0.Slot(uint64(checkpoint.Epoch) * slotsPerEpoch)

	// Check where we are at from the coordinator
	location, err := b.coordinator.GetRelayMonitorLocation(ctx, b.relayMonitorType, b.networkName, b.clientName, b.relayName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get relay monitor location")
	}

	// Default the backfill target slot
	backfillTargetSlot := b.config.MinimumSlot

	// If location is empty, we haven't started yet
	if location == nil {
		// Start from the current finalized slot for head processing
		return &BackfillingSlotNextResponse{
			Next:      finalizedSlot,
			Direction: BackfillingSlotDirectionHead,
		}, nil
	}

	marker, err := b.GetMarker(location)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get marker from location")
	}

	// If the head isn't up to date, return the next finalized slot to process
	if marker.CurrentSlot < uint64(finalizedSlot) {
		next := phase0.Slot(marker.CurrentSlot + 1)

		return &BackfillingSlotNextResponse{
			Next:      next,
			Direction: BackfillingSlotDirectionHead,
		}, nil
	}

	// If backfill is enabled and hasn't completed, return the next backfill slot
	if b.config.Enabled && phase0.Slot(marker.BackfillSlot) > backfillTargetSlot {
		next := phase0.Slot(marker.BackfillSlot - 1)

		b.log.WithFields(logrus.Fields{
			"next_slot":   next,
			"target_slot": backfillTargetSlot,
		}).Debug("Derived next backfill slot to process")

		return &BackfillingSlotNextResponse{
			Next:      next,
			Direction: BackfillingSlotDirectionBackfill,
		}, nil
	}

	// Both head and backfill are complete, return the current slot for head processing
	currentSlot := b.wallclock.Slots().Current()
	return &BackfillingSlotNextResponse{
		Next:      phase0.Slot(currentSlot.Number()),
		Direction: BackfillingSlotDirectionHead,
	}, nil
}

func (b *BackfillingSlot) fetchLatestCheckpoint(_ context.Context) (*phase0.Checkpoint, error) {
	// In a real implementation, this would fetch from the beacon node
	// For now, return a dummy checkpoint
	currentEpoch := b.wallclock.Epochs().Current()
	return &phase0.Checkpoint{
		Epoch: phase0.Epoch(currentEpoch.Number()),
	}, nil
}

func (b *BackfillingSlot) createLocationFromSlotNumber(currentSlot, backfillSlot phase0.Slot) *xatu.RelayMonitorLocation {
	location := &xatu.RelayMonitorLocation{
		MetaNetworkName: b.networkName,
		MetaClientName:  b.clientName,
		RelayName:       b.relayName,
		Type:            b.relayMonitorType,
	}

	marker := &xatu.RelayMonitorSlotMarker{
		CurrentSlot:  uint64(currentSlot),
		BackfillSlot: int64(backfillSlot),
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

func (b *BackfillingSlot) GetMarker(location *xatu.RelayMonitorLocation) (*xatu.RelayMonitorSlotMarker, error) {
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
			return &xatu.RelayMonitorSlotMarker{}, nil
		}
		return data.SlotMarker, nil

	case xatu.RelayMonitorType_RELAY_MONITOR_PAYLOAD_DELIVERED:
		data := location.GetPayloadDelivered()
		if data == nil {
			return nil, errors.New("payload delivered data is nil")
		}
		if data.SlotMarker == nil {
			return &xatu.RelayMonitorSlotMarker{}, nil
		}
		return data.SlotMarker, nil

	default:
		return nil, fmt.Errorf("unknown relay monitor type: %s", b.relayMonitorType)
	}
}
