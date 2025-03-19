package cache

import (
	"context"
	"time"

	"github.com/go-co-op/gocron/v2"
	"github.com/jellydator/ttlcache/v3"
)

type SharedCache struct {
	Transaction *ttlcache.Cache[string, bool]

	metrics *Metrics
}

func NewSharedCache() *SharedCache {
	return &SharedCache{
		Transaction: ttlcache.New(
			ttlcache.WithTTL[string, bool](24 * time.Hour),
		),
		metrics: NewMetrics("xatu_mimicry_coordinator_cache"),
	}
}

func (d *SharedCache) Start(ctx context.Context) error {
	go d.Transaction.Start()

	if err := d.startCrons(ctx); err != nil {
		return err
	}

	return nil
}

func (d *SharedCache) startCrons(ctx context.Context) error {
	c, err := gocron.NewScheduler(gocron.WithLocation(time.Local))
	if err != nil {
		return err
	}

	if _, err := c.NewJob(
		gocron.DurationJob(5*time.Second),
		gocron.NewTask(
			func(ctx context.Context) {
				transactionMetrics := d.Transaction.Metrics()
				d.metrics.SetSharedInsertions(transactionMetrics.Insertions, "transaction")
				d.metrics.SetSharedHits(transactionMetrics.Hits, "transaction")
				d.metrics.SetSharedMisses(transactionMetrics.Misses, "transaction")
				d.metrics.SetSharedEvictions(transactionMetrics.Evictions, "transaction")
			},
			ctx,
		),
	); err != nil {
		return err
	}

	c.Start()

	return nil
}
