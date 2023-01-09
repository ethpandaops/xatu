package cache

import (
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/savid/ttlcache/v3"
)

type SharedCache struct {
	Transaction *ttlcache.Cache[string, *types.Transaction]
}

func NewSharedCache() *SharedCache {
	return &SharedCache{
		Transaction: ttlcache.New(
			ttlcache.WithTTL[string, *types.Transaction](30 * time.Minute),
		),
	}
}

func (d *SharedCache) Start() {
	go d.Transaction.Start()
}
