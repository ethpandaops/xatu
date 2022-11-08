package wallclock

import "time"

type TimeWindow struct {
	start time.Time
	end   time.Time
}

func NewTimeWindow(start, end time.Time) *TimeWindow {
	return &TimeWindow{
		start: start,
		end:   end,
	}
}

func (t *TimeWindow) Start() time.Time {
	return t.start
}

func (t *TimeWindow) End() time.Time {
	return t.end
}

func (t *TimeWindow) Active() bool {
	return t.start.Before(time.Now()) && t.end.After(time.Now())
}

func (t *TimeWindow) EndsIn() time.Duration {
	return time.Until(t.end)
}

func (t *TimeWindow) StartsIn() time.Duration {
	return time.Until(t.start)
}
