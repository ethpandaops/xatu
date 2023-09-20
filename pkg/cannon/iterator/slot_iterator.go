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
	log           logrus.FieldLogger
	cannonType    xatu.CannonType
	coordinator   coordinator.Client
	wallclock     *ethwallclock.EthereumBeaconChain
	networkID     string
	networkName   string
	metrics       *SlotMetrics
	headSlotDelay uint64
}

func NewSlotIterator(log logrus.FieldLogger, networkName, networkID string, cannonType xatu.CannonType, coordinatorClient *coordinator.Client, wallclock *ethwallclock.EthereumBeaconChain, metrics *SlotMetrics, headSlotDelay uint64) *SlotIterator {
	return &SlotIterator{
		log: log.
			WithField("module", "cannon/iterator/slot_iterator").
			WithField("cannon_type", cannonType.String()),
		networkName:   networkName,
		networkID:     networkID,
		cannonType:    cannonType,
		coordinator:   *coordinatorClient,
		wallclock:     wallclock,
		metrics:       metrics,
		headSlotDelay: headSlotDelay,
	}
}

func (s *SlotIterator) UpdateLocation(ctx context.Context, location *xatu.CannonLocation) error {
	return s.coordinator.UpsertCannonLocationRequest(ctx, location)
}

func (s *SlotIterator) Next(ctx context.Context) (next *xatu.CannonLocation, lookAhead []*xatu.CannonLocation, err error) {
	var location *xatu.CannonLocation

	// Calculate the current wallclock slot
	realHeadSlot, _, err := s.wallclock.Now()
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get current wallclock slot")
	}

	defer func() {
		if location != nil {
			slot, err := s.getSlotNumberFromLocation(location)
			if err != nil {
				s.log.WithError(err).Error("failed to get slot number from location")

				return
			}

			s.metrics.SetCurrentSlot(s.cannonType.String(), s.networkName, float64(slot))

			s.metrics.SetTrailingSlots(s.cannonType.String(), s.networkName, float64(realHeadSlot.Number()-uint64(slot)))
		}
	}()

	for {
		// Calculate the current wallclock slot
		realHeadSlot, _, err := s.wallclock.Now()
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get current wallclock slot")
		}

		if realHeadSlot.Number() == 0 {
			return nil, nil, errors.New("network is pre genesis")
		}

		if realHeadSlot.Number() < s.headSlotDelay {
			return nil, nil, errors.New("network is too young")
		}

		// Check where we are at from the coordinator
		location, err = s.coordinator.GetCannonLocation(ctx, s.cannonType, s.networkID)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get cannon location")
		}

		// If location is empty we haven't started yet, start at the network default for the type. If the network default
		// is empty, we'll start at slot 0.
		if location == nil {
			location, err = s.createLocationFromSlotNumber(GetDefaultSlotLocationForNetworkAndType(s.networkName, s.cannonType))
			if err != nil {
				return nil, nil, errors.Wrap(err, "failed to create location from slot number 0")
			}

			return location, nil, nil
		}

		locationSlot, err := s.getSlotNumberFromLocation(location)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get slot number from location")
		}

		// Calculate the maximum slot we should be at
		fakeHeadSlot := phase0.Slot(realHeadSlot.Number() - s.headSlotDelay)

		// Calculate our next slot.
		ourNextSlot := locationSlot + 1

		// Safety check to make sure we aren't too far ahead of the network
		if ourNextSlot >= fakeHeadSlot {
			// Sleep until the wall clock ticks over. If we haven't progressed enough, the next loop iteration will sleep us for
			// nother slot.
			time.Sleep(time.Until(realHeadSlot.TimeWindow().End()))

			continue
		}

		location, err = s.createLocationFromSlotNumber(ourNextSlot)
		if err != nil {
			return nil, nil, errors.Wrap(err, fmt.Errorf("failed to create location from slot number: %d", ourNextSlot).Error())
		}

		return location, nil, nil
	}
}

func (s *SlotIterator) getSlotNumberFromLocation(location *xatu.CannonLocation) (phase0.Slot, error) {
	return 0, errors.Errorf("unknown cannon type %s", location.Type)
}

func (s *SlotIterator) createLocationFromSlotNumber(slot phase0.Slot) (*xatu.CannonLocation, error) {
	location := &xatu.CannonLocation{
		NetworkId: s.networkID,
		Type:      s.cannonType,
	}

	return nil, errors.Errorf("unknown cannon type %s", location.Type)
}
