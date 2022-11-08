package wallclock

import (
	"time"
)

func NewDefaultSlotCreator(genesis time.Time, durationPerSlot time.Duration) *DefaultSlotCreator {
	return &DefaultSlotCreator{
		genesis:         genesis,
		durationPerSlot: durationPerSlot,
	}
}

type DefaultSlotCreator struct {
	genesis         time.Time
	durationPerSlot time.Duration
}

func (s *DefaultSlotCreator) FromNumber(number uint64) Slot {
	return NewSlot(number, s.genesis.Add(time.Duration(number)*s.durationPerSlot), s.genesis.Add(time.Duration(number+1)*s.durationPerSlot))
}

func (s *DefaultSlotCreator) Current() Slot {
	return s.FromTime(time.Now())
}

func (s *DefaultSlotCreator) FromTime(t time.Time) Slot {
	number := uint64(t.Sub(s.genesis) / s.durationPerSlot)
	return s.FromNumber(number)
}
