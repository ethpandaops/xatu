package wallclock

import (
	"time"
)

type Epoch struct {
	number uint64
	window *TimeWindow
}

func NewEpoch(number uint64, start time.Time, end time.Time) Epoch {
	return Epoch{
		number: number,
		window: NewTimeWindow(start, end),
	}
}

func (e *Epoch) Number() uint64 {
	return e.number
}

func (e *Epoch) TimeWindow() *TimeWindow {
	return e.window
}
