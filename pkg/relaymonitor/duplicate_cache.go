package relaymonitor

import (
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec/phase0"
	"github.com/jellydator/ttlcache/v3"
)

// DuplicateBidCache is a cache to store information about whether a relay has seen a block for a specific slot and block hash
type DuplicateBidCache struct {
	ttl    time.Duration
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
	if _, exists := dc.caches[relay]; !exists {
		dc.caches[relay] = ttlcache.New[string, bool](
			ttlcache.WithTTL[string, bool](dc.ttl),
		)

		go dc.caches[relay].Start()
	}

	key := dc.generateKey(slot, blockHash)
	dc.caches[relay].Set(key, true, dc.ttl)
}

// Has checks if a block hash has been seen for a specific relay and slot
func (dc *DuplicateBidCache) Has(relay string, slot phase0.Slot, blockHash string) bool {
	if cache, exists := dc.caches[relay]; exists {
		key := dc.generateKey(slot, blockHash)
		item := cache.Get(key)

		return item != nil && item.Value()
	}

	return false
}

// generateKey creates a unique key for the cache based on slot and blockHash
func (dc *DuplicateBidCache) generateKey(slot phase0.Slot, blockHash string) string {
	return fmt.Sprintf("%d:%s", slot, blockHash)
}
