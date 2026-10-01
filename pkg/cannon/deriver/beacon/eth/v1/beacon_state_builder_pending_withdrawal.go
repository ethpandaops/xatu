package v1

import (
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconStateBuilderPendingWithdrawalDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL
)

type BeaconStateBuilderPendingWithdrawalDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconStateBuilderPendingWithdrawalDeriver snapshots builder_pending_withdrawals each epoch, one event per entry.
type BeaconStateBuilderPendingWithdrawalDeriver struct {
	*gloasStateDeriver
}

func NewBeaconStateBuilderPendingWithdrawalDeriver(log observability.ContextualLogger, config *BeaconStateBuilderPendingWithdrawalDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStateBuilderPendingWithdrawalDeriver {
	d := &BeaconStateBuilderPendingWithdrawalDeriver{}
	d.gloasStateDeriver = newGloasStateDeriver(log, BeaconStateBuilderPendingWithdrawalDeriverName, "beacon_state_builder_pending_withdrawal", config.Enabled, iter, beacon, clientMeta, d.deriveEpoch)

	return d
}

func (d *BeaconStateBuilderPendingWithdrawalDeriver) deriveEpoch(in *gloasStateInput) ([]*xatu.DecoratedEvent, error) {
	epochData := d.epochData(in.epoch)
	events := make([]*xatu.DecoratedEvent, 0, len(in.state.BuilderPendingWithdrawals))

	for position, withdrawal := range in.state.BuilderPendingWithdrawals {
		if withdrawal == nil {
			return nil, fmt.Errorf("builder pending withdrawal %d is nil", position)
		}

		event, err := d.createEvent(uint64(position), withdrawal, epochData, in.stateID)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

func (d *BeaconStateBuilderPendingWithdrawalDeriver) createEvent(position uint64, withdrawal *gloas.BuilderPendingWithdrawal, epoch *xatu.EpochV2, stateID string) (*xatu.DecoratedEvent, error) {
	event, err := d.newEvent(xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER_PENDING_WITHDRAWAL)
	if err != nil {
		return nil, err
	}

	event.Data = &xatu.DecoratedEvent_EthV1BeaconStateBuilderPendingWithdrawal{
		EthV1BeaconStateBuilderPendingWithdrawal: &xatu.BuilderPendingWithdrawalData{
			FeeRecipient: fmt.Sprintf("%#x", withdrawal.FeeRecipient),
			Amount:       wrapperspb.UInt64(uint64(withdrawal.Amount)),
			BuilderIndex: wrapperspb.UInt64(uint64(withdrawal.BuilderIndex)),
		},
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStateBuilderPendingWithdrawal{
		EthV1BeaconStateBuilderPendingWithdrawal: &xatu.ClientMeta_AdditionalEthV1BeaconStateBuilderPendingWithdrawalData{
			Epoch:           epoch,
			StateId:         stateID,
			PositionInQueue: wrapperspb.UInt64(position),
		},
	}

	return event, nil
}
