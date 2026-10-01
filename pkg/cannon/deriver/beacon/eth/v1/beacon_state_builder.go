package v1

import (
	"fmt"

	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/cannon/iterator"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	BeaconStateBuilderDeriverName = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_BUILDER

	builderStatusPending = "pending"
	builderStatusActive  = "active"
	builderStatusExited  = "exited"

	farFutureEpoch = phase0.Epoch(1<<64 - 1)
)

type BeaconStateBuilderDeriverConfig struct {
	Enabled  bool                                 `yaml:"enabled" default:"true"`
	Iterator iterator.BackfillingCheckpointConfig `yaml:"iterator"`
}

// BeaconStateBuilderDeriver snapshots the builder registry each epoch, one event per builder.
type BeaconStateBuilderDeriver struct {
	*gloasStateDeriver
}

func NewBeaconStateBuilderDeriver(log observability.ContextualLogger, config *BeaconStateBuilderDeriverConfig, iter *iterator.BackfillingCheckpoint, beacon *ethereum.BeaconNode, clientMeta *xatu.ClientMeta) *BeaconStateBuilderDeriver {
	d := &BeaconStateBuilderDeriver{}
	d.gloasStateDeriver = newGloasStateDeriver(log, BeaconStateBuilderDeriverName, "beacon_state_builder", config.Enabled, iter, beacon, clientMeta, d.deriveEpoch)

	return d
}

func (d *BeaconStateBuilderDeriver) deriveEpoch(in *gloasStateInput) ([]*xatu.DecoratedEvent, error) {
	epochData := d.epochData(in.epoch)
	events := make([]*xatu.DecoratedEvent, 0, len(in.state.Builders))

	for index, builder := range in.state.Builders {
		if builder == nil {
			return nil, fmt.Errorf("builder %d is nil", index)
		}

		event, err := d.createEvent(uint64(index), builder, builderStatus(builder, in.state.FinalizedEpoch), epochData, in.stateID)
		if err != nil {
			return nil, err
		}

		events = append(events, event)
	}

	return events, nil
}

// builderStatus is the beacon API builder status; active matches the spec's is_active_builder.
func builderStatus(builder *gloas.Builder, finalizedEpoch phase0.Epoch) string {
	switch {
	case builder.WithdrawableEpoch != farFutureEpoch:
		return builderStatusExited
	case builder.DepositEpoch < finalizedEpoch:
		return builderStatusActive
	default:
		return builderStatusPending
	}
}

func (d *BeaconStateBuilderDeriver) createEvent(index uint64, builder *gloas.Builder, status string, epoch *xatu.EpochV2, stateID string) (*xatu.DecoratedEvent, error) {
	event, err := d.newEvent(xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_BUILDER)
	if err != nil {
		return nil, err
	}

	event.Data = &xatu.DecoratedEvent_EthV1BeaconStateBuilder{
		EthV1BeaconStateBuilder: &xatu.BuilderData{
			Index:             wrapperspb.UInt64(index),
			Pubkey:            fmt.Sprintf("%#x", builder.PublicKey),
			Version:           wrapperspb.UInt32(uint32(builder.Version)),
			ExecutionAddress:  fmt.Sprintf("%#x", builder.ExecutionAddress),
			Balance:           wrapperspb.UInt64(uint64(builder.Balance)),
			DepositEpoch:      wrapperspb.UInt64(uint64(builder.DepositEpoch)),
			WithdrawableEpoch: wrapperspb.UInt64(uint64(builder.WithdrawableEpoch)),
			Status:            status,
		},
	}

	event.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV1BeaconStateBuilder{
		EthV1BeaconStateBuilder: &xatu.ClientMeta_AdditionalEthV1BeaconStateBuilderData{
			Epoch:   epoch,
			StateId: stateID,
		},
	}

	return event, nil
}
