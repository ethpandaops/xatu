package telemetry

import (
	"sync"
	"time"
)

// LogSampler gates log emission so that at most one log is emitted per
// interval for each unique key. Callers use Allow() to check whether a
// log should be written; suppressed occurrences are counted and reported
// when the next log is allowed.
//
// Keys should be bounded (e.g. event names, table names, topic names)
// so that the internal map does not grow unboundedly.
type LogSampler struct {
	interval time.Duration
	mu       sync.Mutex
	entries  map[string]*sampledEntry
}

type sampledEntry struct {
	lastEmit   time.Time
	suppressed int64
}

// NewLogSampler creates a sampler that allows one log per interval per key.
func NewLogSampler(interval time.Duration) *LogSampler {
	return &LogSampler{
		interval: interval,
		entries:  make(map[string]*sampledEntry),
	}
}

// Allow returns whether a log should be emitted for the given key, and
// the number of occurrences suppressed since the last emission.
func (s *LogSampler) Allow(key string) (bool, int64) {
	now := time.Now()

	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.entries[key]
	if !ok {
		s.entries[key] = &sampledEntry{lastEmit: now}

		return true, 0
	}

	if now.Sub(entry.lastEmit) < s.interval {
		entry.suppressed++

		return false, 0
	}

	suppressed := entry.suppressed
	entry.suppressed = 0
	entry.lastEmit = now

	return true, suppressed
}
