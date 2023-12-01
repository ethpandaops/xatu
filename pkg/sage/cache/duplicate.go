package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
)

type DuplicateCache struct {
	Attestation *ttlcache.Cache[string, int]
}

const (
	// best to keep this > 1 epoch as some clients may send the same attestation on new epoch
	TTL = 7 * time.Minute
)

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		Attestation: ttlcache.New(
			ttlcache.WithTTL[string, int](TTL),
		),
	}
}

func (d *DuplicateCache) Start() {
	go d.Attestation.Start()
}
