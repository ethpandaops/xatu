package cache

import (
	"time"

	"github.com/savid/ttlcache/v3"
)

type DuplicateCache struct {
	Node *ttlcache.Cache[string, time.Time]
}

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		Node: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](120 * time.Minute),
		),
	}
}

func (d *DuplicateCache) Start() {
	go d.Node.Start()
}
