package iterator

import (
	"errors"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/ethpandaops/beacon/pkg/beacon/state"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

var (
	ErrLocationUpToDate = errors.New("location up to date")

	SlotZero = phase0.Slot(0)
)

func GetDefaultSlotLocation(forkEpochs state.ForkEpochs, cannonType xatu.CannonType) phase0.Slot {
	defaults := NewSlotDefaultsFromForkEpochs(forkEpochs)
	if slot, exists := defaults[cannonType]; exists {
		return slot
	}

	return SlotZero
}

func GetStartingEpochLocation(forkEpochs state.ForkEpochs, cannonType xatu.CannonType) phase0.Epoch {
	defaults := NewSlotDefaultsFromForkEpochs(forkEpochs)
	if slot, exists := defaults[cannonType]; exists {
		return phase0.Epoch(slot / 32)
	}

	return 0
}
