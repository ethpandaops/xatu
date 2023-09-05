package iterator

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/cannon/coordinator"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type SlotIterator struct {
	log         logrus.FieldLogger
	cannonType  xatu.CannonType
	coordinator coordinator.Client
	wallclock   *ethwallclock.EthereumBeaconChain
	networkID   string
}

func NewSlotIterator(log logrus.FieldLogger, networkID string, cannonType xatu.CannonType, coordinatorClient *coordinator.Client, wallclock *ethwallclock.EthereumBeaconChain) *SlotIterator {
	return &SlotIterator{
		log: log.
			WithField("module", "cannon/iterator/slot_iterator").
			WithField("cannon_type", cannonType.String()),
		networkID:   networkID,
		cannonType:  cannonType,
		coordinator: *coordinatorClient,
		wallclock:   wallclock,
	}
}

func (s *SlotIterator) UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error {
	return s.coordinator.UpsertCannonLocationRequest(ctx, location)
}

func (s *SlotIterator) Next(ctx context.Context) (*xatu.CannonLocation, error) {
	// Check where we are at from the coordinator
	location, err := s.coordinator.GetCannonLocation(ctx, s.cannonType, s.networkID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cannon location")
	}

	// If location is empty we haven't started yet, start at slot 0
	if location == nil {
		loc, errr := s.createLocationFromSlotNumber(0)
		if errr != nil {
			return nil, errors.Wrap(err, "failed to create location from slot number 0")
		}

		return loc, nil
	}

	// Calculate the current wallclock slot
	headSlot, _, err := s.wallclock.Now()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current wallclock slot")
	}

	locationSlot, err := s.getSlotNumberFromLocation(location)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get slot number from location")
	}

	// If the current wallclock slot is greater than the last slot we processed, return the next slot
	if phase0.Slot(headSlot.Number()) > locationSlot {
		loc, errr := s.createLocationFromSlotNumber(locationSlot + 1)
		if errr != nil {
			return nil, errors.Wrap(errr, fmt.Errorf("failed to create location from slot number: %d", locationSlot+1).Error())
		}

		return loc, nil
	}

	// Sleep until the next slot
	sleepTime := time.Until(headSlot.TimeWindow().Start())

	if sleepTime.Milliseconds() > 0 {
		s.log.WithField("sleep_time", sleepTime).Debug("sleeping until next slot")

		time.Sleep(sleepTime)
	}

	loc, err := s.createLocationFromSlotNumber(phase0.Slot(headSlot.Number()))
	if err != nil {
		return nil, errors.Wrap(err, fmt.Errorf("failed to create location from slot number: %d", headSlot.Number()).Error())
	}

	return loc, nil
}

func (s *SlotIterator) getSlotNumberFromLocation(location *xatu.CannonLocation) (phase0.Slot, error) {
	switch location.Type {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		return phase0.Slot(location.GetEthV2BeaconBlockAttesterSlashing().Slot), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		return phase0.Slot(location.GetEthV2BeaconBlockProposerSlashing().Slot), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		return phase0.Slot(location.GetEthV2BeaconBlockBlsToExecutionChange().Slot), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		return phase0.Slot(location.GetEthV2BeaconBlockExecutionTransaction().Slot), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		return phase0.Slot(location.GetEthV2BeaconBlockVoluntaryExit().Slot), nil
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		return phase0.Slot(location.GetEthV2BeaconBlockDeposit().Slot), nil
	default:
		return 0, errors.Errorf("unknown cannon type %s", location.Type)
	}
}

func (s *SlotIterator) createLocationFromSlotNumber(slot phase0.Slot) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: s.networkID,
		Type:      s.cannonType,
	}

	switch s.cannonType {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: &xatu.CannonLocationEthV2BeaconBlockAttesterSlashing{
				Slot: uint64(slot),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: &xatu.CannonLocationEthV2BeaconBlockProposerSlashing{
				Slot: uint64(slot),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: &xatu.CannonLocationEthV2BeaconBlockBlsToExecutionChange{
				Slot: uint64(slot),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: &xatu.CannonLocationEthV2BeaconBlockExecutionTransaction{
				Slot: uint64(slot),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: &xatu.CannonLocationEthV2BeaconBlockVoluntaryExit{
				Slot: uint64(slot),
			},
		}
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		location.Data = &xatu.CannonLocation_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: &xatu.CannonLocationEthV2BeaconBlockDeposit{
				Slot: uint64(slot),
			},
		}
	default:
		return location, errors.Errorf("unknown cannon type %s", location.Type)
	}

	return location, nil
}
