package cache

import (
	"time"

	"github.com/jellydator/ttlcache/v3"
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
	BeaconEthV1BeaconCommittee            *ttlcache.Cache[string, time.Time]
	BeaconEthV1EventsBlobSidecar          *ttlcache.Cache[string, time.Time]
}

const (
	// best to keep this > 1 epoch as some clients may send the same attestation on new epoch
	TTL = 7 * time.Minute
)

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		BeaconETHV1EventsAttestation: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV1EventsBlock: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV1EventsChainReorg: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV1EventsFinalizedCheckpoint: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV1EventsHead: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV1EventsVoluntaryExit: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV1EventsContributionAndProof: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconETHV2BeaconBlock: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconEthV1BeaconCommittee: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
		),
		BeaconEthV1EventsBlobSidecar: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](TTL),
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
	go d.BeaconEthV1BeaconCommittee.Start()
	go d.BeaconEthV1EventsBlobSidecar.Start()
}
