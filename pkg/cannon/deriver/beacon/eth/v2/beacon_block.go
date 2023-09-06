package v2

import (
	"fmt"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/ethpandaops/ethwallclock"
	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func GetBlockIdentifier(block *spec.VersionedSignedBeaconBlock, wallclock *ethwallclock.EthereumBeaconChain) (*xatu.BlockIdentifier, error) {
	if block == nil {
		return nil, fmt.Errorf("block is nil")
	}

	slotNum, err := block.Slot()
	if err != nil {
		return nil, err
	}

	root, err := block.Root()
	if err != nil {
		return nil, err
	}

	slot := wallclock.Slots().FromNumber(uint64(slotNum))
	epoch := wallclock.Epochs().FromSlot(uint64(slotNum))

	return &xatu.BlockIdentifier{
		Epoch: &xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		},
		Slot: &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: slot.Number()},
			StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		},
		Root:    v1.RootAsString(root),
		Version: block.Version.String(),
	}, nil
}
