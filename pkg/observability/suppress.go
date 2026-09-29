package observability

import (
	"sync/atomic"
	"time"
)

// Suppressor rate-limits a repeated log line to one emission per Window and
// counts what was held back in between. It is intended for per-event error
// paths whose failure volume is unbounded: the first failure is logged in
// full, later ones only increment a counter, and the next allowed call
// reports how many were suppressed so the log stays self-sufficient.
//
// The suppressed path is lock-free and costs one atomic load and one atomic
// add. A Suppressor must not be copied after first use.
type Suppressor struct {
	// Window is the minimum interval between allowed calls.
	Window time.Duration

	nextAllow  atomic.Int64  // unix nanos before which calls are suppressed
	suppressed atomic.Uint64 // calls suppressed since the last allowed one
}

// NewSuppressors returns n Suppressors sharing the same window, one per
// element of a parallel slice such as a list of sinks.
func NewSuppressors(n int, window time.Duration) []Suppressor {
	suppressors := make([]Suppressor, n)
	for i := range suppressors {
		suppressors[i].Window = window
	}

	return suppressors
}

// Allow reports whether the caller should log now and how many calls were
// suppressed since the last allowed one. The first call always succeeds.
func (s *Suppressor) Allow(now time.Time) (allowed bool, suppressed uint64) {
	next := s.nextAllow.Load()
	if now.UnixNano() < next {
		s.suppressed.Add(1)

		return false, 0
	}

	if !s.nextAllow.CompareAndSwap(next, now.Add(s.Window).UnixNano()) {
		s.suppressed.Add(1)

		return false, 0
	}

	return true, s.suppressed.Swap(0)
}
