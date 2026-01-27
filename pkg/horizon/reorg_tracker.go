package horizon

import (
	"sync"
	"time"
)

// ReorgTracker tracks slots that were affected by reorgs to annotate derived events.
type ReorgTracker struct {
	mu    sync.Mutex
	slots map[uint64]time.Time
	ttl   time.Duration
}

// NewReorgTracker creates a new tracker with the given TTL.
func NewReorgTracker(ttl time.Duration) *ReorgTracker {
	if ttl <= 0 {
		ttl = 13 * time.Minute
	}

	return &ReorgTracker{
		slots: make(map[uint64]time.Time),
		ttl:   ttl,
	}
}

// AddRange marks slots in [start, end] as affected by a reorg.
func (r *ReorgTracker) AddRange(start, end uint64) {
	if end < start {
		return
	}

	now := time.Now()
	expiry := now.Add(r.ttl)

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanupLocked(now)

	for slot := start; slot <= end; slot++ {
		r.slots[slot] = expiry
	}
}

// IsReorgSlot reports whether a slot is marked as reorg-affected.
func (r *ReorgTracker) IsReorgSlot(slot uint64) bool {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanupLocked(now)

	expiry, ok := r.slots[slot]
	if !ok {
		return false
	}

	if now.After(expiry) {
		delete(r.slots, slot)
		return false
	}

	return true
}

func (r *ReorgTracker) cleanupLocked(now time.Time) {
	for slot, expiry := range r.slots {
		if now.After(expiry) {
			delete(r.slots, slot)
		}
	}
}
