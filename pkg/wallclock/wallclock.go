package wallclock

import (
	"time"
)

type EthereumBeaconChain struct {
	slots  *DefaultSlotCreator
	epochs *DefaultEpochCreator
}

func NewEthereumBeaconChain(genesis time.Time, durationPerSlot time.Duration, slotsPerEpoch uint64) *EthereumBeaconChain {
	return &EthereumBeaconChain{
		slots:  NewDefaultSlotCreator(genesis, durationPerSlot),
		epochs: NewDefaultEpochCreator(genesis, durationPerSlot, slotsPerEpoch),
	}
}

func (e *EthereumBeaconChain) Now() (Slot, Epoch, error) {
	slot := e.slots.Current()
	epoch := e.epochs.Current()
	return slot, epoch, nil
}

func (e *EthereumBeaconChain) FromTime(t time.Time) (Slot, Epoch, error) {
	slot := e.slots.FromTime(t)
	epoch := e.epochs.FromTime(t)
	return slot, epoch, nil
}

func (e *EthereumBeaconChain) Slots() *DefaultSlotCreator {
	return e.slots
}

func (e *EthereumBeaconChain) Epochs() *DefaultEpochCreator {
	return e.epochs
}
