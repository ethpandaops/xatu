package cache

import (
	"time"

	"github.com/savid/ttlcache/v3"
)

type DuplicateCache struct {
	Transaction *ttlcache.Cache[string, time.Time]
}

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		Transaction: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
	}
}

func (d *DuplicateCache) Start() {
	go d.Transaction.Start()
}
