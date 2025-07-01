package cache

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jellydator/ttlcache/v3"
)

type DuplicateCache struct {
	Node      *ttlcache.Cache[string, time.Time]
	metrics   *Metrics
	scheduler gocron.Scheduler
}

func NewDuplicateCache() *DuplicateCache {
	return &DuplicateCache{
		Node: ttlcache.New(
			ttlcache.WithTTL[string, time.Time](120 * time.Minute),
		),
		metrics: NewMetrics("xatu_discovery_cache"),
	}
}

func (d *DuplicateCache) Start(ctx context.Context) error {
	go d.Node.Start()

	if err := d.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

func (d *DuplicateCache) Stop() {
	d.Node.Stop()

	if d.scheduler != nil {
		_ = d.scheduler.Shutdown()
	}
}

func (d *DuplicateCache) startCrons(ctx context.Context) error {
	scheduler, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	d.scheduler = scheduler

	if _, err := d.scheduler.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			func(ctx context.Context) {
				nodeMetrics := d.Node.Metrics()
				d.metrics.SetDuplicateInsertions(nodeMetrics.Insertions, "node")
				d.metrics.SetDuplicateHits(nodeMetrics.Hits, "node")
				d.metrics.SetDuplicateMisses(nodeMetrics.Misses, "node")
				d.metrics.SetDuplicateEvictions(nodeMetrics.Evictions, "node")
			},
			ctx,
		),
		gocron.WithStartAt(gocron.WithStartImmediately()),
	); err != nil {
		return err
	}

	d.scheduler.Start()

	return nil
}
