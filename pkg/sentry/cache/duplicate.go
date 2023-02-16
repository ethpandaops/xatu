package cache

import (
	"time"

	"github.com/savid/ttlcache/v3"
)

type DuplicateCache struct {
	BeaconETHV1EventsAttestation          *ttlcache.Cache[string, time.Time]
	BeaconETHV1EventsBlock                *ttlcache.Cache[string, time.Time]
	BeaconETHV1EventsChainReorg           *ttlcache.Cache[string, time.Time]
	BeaconETHV1EventsFinalizedCheckpoint  *ttlcache.Cache[string, time.Time]
	BeaconETHV1EventsHead                 *ttlcache.Cache[string, time.Time]
	BeaconETHV1EventsVoluntaryExit        *ttlcache.Cache[string, time.Time]
	BeaconETHV1EventsContributionAndProof *ttlcache.Cache[string, time.Time]
	BeaconETHV2BeaconBlock                *ttlcache.Cache[string, time.Time]
}

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		BeaconETHV1EventsAttestation: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV1EventsBlock: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV1EventsChainReorg: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV1EventsFinalizedCheckpoint: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV1EventsHead: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV1EventsVoluntaryExit: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV1EventsContributionAndProof: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
		BeaconETHV2BeaconBlock: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](30 * time.Minute),
		),
	}
}

func (d *DuplicateCache) Start() {
	go d.BeaconETHV1EventsAttestation.Start()
	go d.BeaconETHV1EventsBlock.Start()
	go d.BeaconETHV1EventsChainReorg.Start()
	go d.BeaconETHV1EventsFinalizedCheckpoint.Start()
	go d.BeaconETHV1EventsHead.Start()
	go d.BeaconETHV1EventsVoluntaryExit.Start()
	go d.BeaconETHV1EventsContributionAndProof.Start()
	go d.BeaconETHV2BeaconBlock.Start()
}
