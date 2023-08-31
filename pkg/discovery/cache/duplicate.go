package cache

import (
	"context"
	"time"

	"github.com/go-co-op/gocron"
	"github.com/jellydator/ttlcache/v3"
)

type DuplicateCache struct {
	Node *ttlcache.Cache[string, time.Time]

	metrics *Metrics
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
}

func (d *DuplicateCache) startCrons(ctx context.Context) error {
	c := gocron.NewScheduler(time.Local)

	if _, err := c.Every("5s").Do(func() {
		nodeMetrics := d.Node.Metrics()
		d.metrics.SetDuplicateInsertions(nodeMetrics.Insertions, "node")
		d.metrics.SetDuplicateHits(nodeMetrics.Hits, "node")
		d.metrics.SetDuplicateMisses(nodeMetrics.Misses, "node")
		d.metrics.SetDuplicateEvictions(nodeMetrics.Evictions, "node")
	}); err != nil {
		return err
	}

	c.StartAsync()

	return nil
}
