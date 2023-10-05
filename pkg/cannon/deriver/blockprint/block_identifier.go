package blockprint

import (
	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/ethwallclock"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func GetSlotEpochTimings(slot phase0.Slot, wallclock *ethwallclock.EthereumBeaconChain) (*xatu.SlotV2, *xatu.EpochV2) {
	slotNum := wallclock.Slots().FromNumber(uint64(slot))
	epoch := wallclock.Epochs().FromSlot(uint64(slot))

	return &xatu.SlotV2{
			Number:        &wrapperspb.UInt64Value{Value: slotNum.Number()},
			StartDateTime: timestamppb.New(slotNum.TimeWindow().Start()),
		},
		&xatu.EpochV2{
			Number:        &wrapperspb.UInt64Value{Value: epoch.Number()},
			StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
		}
}
