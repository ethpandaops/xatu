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
	BeaconStatePtcMemberDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER
)

type BeaconStatePtcMemberDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconStatePtcMemberDeriver emits the epoch's ordered payload timeliness committees from ptc_window, one event per position.
// Position is the bit index in PayloadAttestation.aggregation_bits, and a validator can hold several positions.
type BeaconStatePtcMemberDeriver struct {
	*gloasStateDeriver
}

func NewBeaconStatePtcMemberDeriver(log observability.ContextualLogger, config *BeaconStatePtcMemberDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStatePtcMemberDeriver {
	d := &BeaconStatePtcMemberDeriver{}
	d.gloasStateDeriver = newGloasStateDeriver(log, BeaconStatePtcMemberDeriverName, "beacon_state_ptc_member", config.Enabled, iter, beacon, clientMeta, d.deriveEpoch)

	return d
}

func (d *BeaconStatePtcMemberDeriver) deriveEpoch(in *gloasStateInput) ([]*xatu.DecoratedEvent, error) {
	firstSlot := phase0.Slot(uint64(in.epoch) * in.slotsPerEpoch)
	epochData := d.epochData(in.epoch)

	var events []*xatu.DecoratedEvent

	for slot := firstSlot; slot < firstSlot+phase0.Slot(in.slotsPerEpoch); slot++ {
		committee, err := in.state.PTC(slot, in.slotsPerEpoch)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to read payload timeliness committee of slot %d", slot)
		}

		if len(committee) == 0 {
			return nil, errors.Errorf("payload timeliness committee of slot %d is empty", slot)
		}

		if events == nil {
			events = make([]*xatu.DecoratedEvent, 0, int(in.slotsPerEpoch)*len(committee)) //nolint:gosec // slots per epoch is small
		}

		slotData := d.slotData(slot)

		for position, validatorIndex := range committee {
			event, err := d.createEvent(position, validatorIndex, slotData, epochData, in.stateID)
			if err != nil {
				return nil, err
			}

			events = append(events, event)
		}
	}

	return events, nil
}

func (d *BeaconStatePtcMemberDeriver) createEvent(position int, validatorIndex phase0.ValidatorIndex, slot *xatu.SlotV2, epoch *xatu.EpochV2, stateID string) (*xatu.DecoratedEvent, error) {
	event, err := d.newEvent(xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER)
	if err != nil {
		return nil, err
	}

	event.Data = &xatu.DecoratedEvent_EthV1BeaconStatePtcMember{
		EthV1BeaconStatePtcMember: &xatu.PtcMemberData{
			Position:       wrapperspb.UInt64(uint64(position)), //nolint:gosec // position is a small index
			ValidatorIndex: wrapperspb.UInt64(uint64(validatorIndex)),
		},
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStatePtcMember{
		EthV1BeaconStatePtcMember: &xatu.ClientMeta_AdditionalEthV1BeaconStatePtcMemberData{
			Slot:    slot,
			Epoch:   epoch,
			StateId: stateID,
		},
	}

	return event, nil
}
