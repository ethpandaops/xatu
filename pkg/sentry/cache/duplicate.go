package cache

import (
	"time"

	"github.com/savid/ttlcache/v3"
)

type DuplicateCache struct {
	Attestation          *ttlcache.Cache[string, time.Time]
	Block                *ttlcache.Cache[string, time.Time]
	ChainReorg           *ttlcache.Cache[string, time.Time]
	FinalizedCheckpoint  *ttlcache.Cache[string, time.Time]
	Head                 *ttlcache.Cache[string, time.Time]
	VoluntaryExit        *ttlcache.Cache[string, time.Time]
	ContributionAndProof *ttlcache.Cache[string, time.Time]
}

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		Attestation: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		Block: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		ChainReorg: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		FinalizedCheckpoint: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		Head: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		VoluntaryExit: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		ContributionAndProof: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
	}
}

func (d *DuplicateCache) Start() {
	go d.Attestation.Start()
	go d.Block.Start()
	go d.ChainReorg.Start()
	go d.FinalizedCheckpoint.Start()
	go d.Head.Start()
	go d.VoluntaryExit.Start()
	go d.ContributionAndProof.Start()
}
