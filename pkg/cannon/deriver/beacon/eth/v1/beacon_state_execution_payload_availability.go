package v1

import (
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconStateExecutionPayloadAvailabilityDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY

	// availabilityEpochLag is how far behind the state's epoch the emitted slots are, so their bits are settled.
	availabilityEpochLag = 2
)

type BeaconStateExecutionPayloadAvailabilityDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconStateExecutionPayloadAvailabilityDeriver emits each slot's availability bit, reading epoch E-2's slots from the state at epoch E.
// The first Gloas slot is skipped because the fork upgrade sets every bit and never clears that one.
type BeaconStateExecutionPayloadAvailabilityDeriver struct {
	*gloasStateDeriver
}

func NewBeaconStateExecutionPayloadAvailabilityDeriver(log observability.ContextualLogger, config *BeaconStateExecutionPayloadAvailabilityDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStateExecutionPayloadAvailabilityDeriver {
	d := &BeaconStateExecutionPayloadAvailabilityDeriver{}
	d.gloasStateDeriver = newGloasStateDeriver(log, BeaconStateExecutionPayloadAvailabilityDeriverName, "beacon_state_execution_payload_availability", config.Enabled, iter, beacon, clientMeta, d.deriveEpoch)

	return d
}

func (d *BeaconStateExecutionPayloadAvailabilityDeriver) deriveEpoch(in *gloasStateInput) ([]*xatu.DecoratedEvent, error) {
	if in.epoch < availabilityEpochLag {
		return nil, nil
	}

	slotEpoch := in.epoch - availabilityEpochLag
	if slotEpoch < in.forkEpoch {
		return nil, nil
	}

	firstGloasSlot := phase0.Slot(uint64(in.forkEpoch) * in.slotsPerEpoch)
	firstSlot := phase0.Slot(uint64(slotEpoch) * in.slotsPerEpoch)
	epochData := d.epochData(slotEpoch)
	events := make([]*xatu.DecoratedEvent, 0, in.slotsPerEpoch)

	for slot := firstSlot; slot < firstSlot+phase0.Slot(in.slotsPerEpoch); slot++ {
		if slot <= firstGloasSlot {
			continue
		}

		if !in.state.PayloadAvailabilitySettled(slot) {
			return nil, errors.Errorf("execution payload availability of slot %d is not settled in the state at slot %d", slot, in.state.Slot)
		}

		available, err := in.state.PayloadAvailable(slot)
		if err != nil {
			return nil, err
		}

		event, err := d.createEvent(available, d.slotData(slot), epochData, in.stateID)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

func (d *BeaconStateExecutionPayloadAvailabilityDeriver) createEvent(available bool, slot *xatu.SlotV2, epoch *xatu.EpochV2, stateID string) (*xatu.DecoratedEvent, error) {
	event, err := d.newEvent(xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_EXECUTION_PAYLOAD_AVAILABILITY)
	if err != nil {
		return nil, err
	}

	event.Data = &xatu.DecoratedEvent_EthV1BeaconStateExecutionPayloadAvailability{
		EthV1BeaconStateExecutionPayloadAvailability: &xatu.ExecutionPayloadAvailabilityData{
			Available: wrapperspb.Bool(available),
		},
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStateExecutionPayloadAvailability{
		EthV1BeaconStateExecutionPayloadAvailability: &xatu.ClientMeta_AdditionalEthV1BeaconStateExecutionPayloadAvailabilityData{
			Slot:    slot,
			Epoch:   epoch,
			StateId: stateID,
		},
	}

	return event, nil
}
