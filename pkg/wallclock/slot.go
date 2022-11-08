package wallclock

import (
	"time"
)

type Slot struct {
	number uint64
	window *TimeWindow
}

func NewSlot(number uint64, start, end time.Time) Slot {
	return Slot{
		number: number,
		window: NewTimeWindow(start, end),
	}
}

func (s *Slot) Number() uint64 {
	return s.number
}

func (s *Slot) TimeWindow() *TimeWindow {
	return s.window
}
