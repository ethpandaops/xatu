package wallclock

import (
	"time"
)

func NewDefaultEpochCreator(genesis time.Time, durationPerSlot time.Duration, slotsPerEpoch uint64) *DefaultEpochCreator {
	return &DefaultEpochCreator{
		genesis:         genesis,
		durationPerSlot: durationPerSlot,
		slotsPerEpoch:   slotsPerEpoch,
	}
}

type DefaultEpochCreator struct {
	genesis         time.Time
	durationPerSlot time.Duration
	slotsPerEpoch   uint64
}

func (e *DefaultEpochCreator) FromNumber(number uint64) Epoch {
	return NewEpoch(number,
		e.genesis.Add(time.Duration(number*e.slotsPerEpoch)*e.durationPerSlot),
		e.genesis.Add(time.Duration((number+1)*e.slotsPerEpoch)*e.durationPerSlot),
	)
}

func (e *DefaultEpochCreator) Current() Epoch {
	return e.FromTime(time.Now())
}

func (e *DefaultEpochCreator) FromTime(t time.Time) Epoch {
	number := uint64(t.Sub(e.genesis) / (e.durationPerSlot * time.Duration(e.slotsPerEpoch)))
	return e.FromNumber(number)
}
