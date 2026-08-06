package relaymonitor

import (
	"fmt"
	"sync"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/jellydator/ttlcache/v3"
)

// DuplicateBidCache is a cache to store information about whether a relay has seen a block for a specific slot and block hash
type DuplicateBidCache struct {
	ttl time.Duration

	mu     sync.RWMutex
	caches map[string]*ttlcache.Cache[string, bool]
}

// NewDuplicateBidCache creates a new DuplicateBidCache with a specified TTL
func NewDuplicateBidCache(ttl time.Duration) *DuplicateBidCache {
	return &DuplicateBidCache{
		caches: make(map[string]*ttlcache.Cache[string, bool]),
		ttl:    ttl,
	}
}

// Set marks a block hash as seen for a specific relay and slot
func (dc *DuplicateBidCache) Set(relay string, slot phase0.Slot, blockHash string) {
	cache := dc.getOrCreateCache(relay)

	key := dc.generateKey(slot, blockHash)
	cache.Set(key, true, dc.ttl)
}

// Has checks if a block hash has been seen for a specific relay and slot
func (dc *DuplicateBidCache) Has(relay string, slot phase0.Slot, blockHash string) bool {
	dc.mu.RLock()
	cache, exists := dc.caches[relay]
	dc.mu.RUnlock()

	if !exists {
		return false
	}

	key := dc.generateKey(slot, blockHash)
	item := cache.Get(key)

	return item != nil && item.Value()
}

// getOrCreateCache returns the per-relay cache, creating and starting it on
// first use. Callers for a given relay run concurrently (one goroutine per
// configured slot-time offset, plus the consistency coordinator), so the
// lookup, creation and insertion into the shared map all need to happen
// under the same lock.
func (dc *DuplicateBidCache) getOrCreateCache(relay string) *ttlcache.Cache[string, bool] {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	cache, exists := dc.caches[relay]
	if !exists {
		cache = ttlcache.New[string, bool](
			ttlcache.WithTTL[string, bool](dc.ttl),
		)
		dc.caches[relay] = cache

		go cache.Start()
	}

	return cache
}

// generateKey creates a unique key for the cache based on slot and blockHash
func (dc *DuplicateBidCache) generateKey(slot phase0.Slot, blockHash string) string {
	return fmt.Sprintf("%d:%s", slot, blockHash)
}
