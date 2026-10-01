package v1

import (
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconStateBuilderPendingPaymentDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT
)

type BeaconStateBuilderPendingPaymentDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconStateBuilderPendingPaymentDeriver snapshots builder_pending_payments each epoch, skipping entries equal to BuilderPendingPayment.empty().
type BeaconStateBuilderPendingPaymentDeriver struct {
	*gloasStateDeriver
}

func NewBeaconStateBuilderPendingPaymentDeriver(log observability.ContextualLogger, config *BeaconStateBuilderPendingPaymentDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStateBuilderPendingPaymentDeriver {
	d := &BeaconStateBuilderPendingPaymentDeriver{}
	d.gloasStateDeriver = newGloasStateDeriver(log, BeaconStateBuilderPendingPaymentDeriverName, "beacon_state_builder_pending_payment", config.Enabled, iter, beacon, clientMeta, d.deriveEpoch)

	return d
}

func (d *BeaconStateBuilderPendingPaymentDeriver) deriveEpoch(in *gloasStateInput) ([]*xatu.DecoratedEvent, error) {
	payments := in.state.BuilderPendingPayments
	if uint64(len(payments)) != 2*in.slotsPerEpoch {
		return nil, errors.Errorf("state has %d builder pending payments, expected %d", len(payments), 2*in.slotsPerEpoch)
	}

	epochData := d.epochData(in.epoch)

	var events []*xatu.DecoratedEvent

	for index, payment := range payments {
		if payment == nil || payment.Withdrawal == nil {
			return nil, fmt.Errorf("builder pending payment %d is incomplete", index)
		}

		if isEmptyBuilderPendingPayment(payment) {
			continue
		}

		slot, ok := builderPendingPaymentSlot(in.epoch, uint64(index), in.slotsPerEpoch)
		if !ok {
			return nil, fmt.Errorf("builder pending payment %d precedes the first epoch", index)
		}

		event, err := d.createEvent(uint64(index), payment, d.slotData(slot), epochData, in.stateID)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// isEmptyBuilderPendingPayment reports whether payment is BuilderPendingPayment.empty().
func isEmptyBuilderPendingPayment(payment *gloas.BuilderPendingPayment) bool {
	return payment.Weight == 0 &&
		payment.ProposerIndex == 0 &&
		payment.Withdrawal.Amount == 0 &&
		payment.Withdrawal.BuilderIndex == 0 &&
		payment.Withdrawal.FeeRecipient == [20]byte{}
}

// builderPendingPaymentSlot maps an entry to its slot: the first SLOTS_PER_EPOCH entries are the previous epoch's.
func builderPendingPaymentSlot(epoch phase0.Epoch, index, slotsPerEpoch uint64) (phase0.Slot, bool) {
	if index >= slotsPerEpoch {
		return phase0.Slot(uint64(epoch)*slotsPerEpoch + index - slotsPerEpoch), true
	}

	if epoch == 0 {
		return 0, false
	}

	return phase0.Slot((uint64(epoch)-1)*slotsPerEpoch + index), true
}

func (d *BeaconStateBuilderPendingPaymentDeriver) createEvent(index uint64, payment *gloas.BuilderPendingPayment, slot *xatu.SlotV2, epoch *xatu.EpochV2, stateID string) (*xatu.DecoratedEvent, error) {
	event, err := d.newEvent(xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_PAYMENT)
	if err != nil {
		return nil, err
	}

	event.Data = &xatu.DecoratedEvent_EthV1BeaconStateBuilderPendingPayment{
		EthV1BeaconStateBuilderPendingPayment: &xatu.BuilderPendingPaymentData{
			PaymentIndex:  wrapperspb.UInt64(index),
			Weight:        wrapperspb.UInt64(uint64(payment.Weight)),
			FeeRecipient:  fmt.Sprintf("%#x", payment.Withdrawal.FeeRecipient),
			Amount:        wrapperspb.UInt64(uint64(payment.Withdrawal.Amount)),
			BuilderIndex:  wrapperspb.UInt64(uint64(payment.Withdrawal.BuilderIndex)),
			ProposerIndex: wrapperspb.UInt64(uint64(payment.ProposerIndex)),
		},
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStateBuilderPendingPayment{
		EthV1BeaconStateBuilderPendingPayment: &xatu.ClientMeta_AdditionalEthV1BeaconStateBuilderPendingPaymentData{
			Epoch:   epoch,
			Slot:    slot,
			StateId: stateID,
		},
	}

	return event, nil
}
