package clickhouse

import "time"

func (w *ChGoWriter) collectPoolMetrics() {
	ticker := time.NewTicker(w.chgoCfg.PoolMetricsInterval)
	defer ticker.Stop()

	var prevAcquireCount int64

	var prevEmptyAcquireCount int64

	var prevCanceledAcquireCount int64

	for {
		select {
		case <-w.poolMetricsDone:
			return
		case <-ticker.C:
			pool := w.getPool()
			if pool == nil {
				continue
			}

			stat := pool.Stat()

			w.metrics.ChgoPoolAcquiredResources().Set(float64(stat.AcquiredResources()))
			w.metrics.ChgoPoolIdleResources().Set(float64(stat.IdleResources()))
			w.metrics.ChgoPoolConstructingResources().Set(float64(stat.ConstructingResources()))
			w.metrics.ChgoPoolTotalResources().Set(float64(stat.TotalResources()))
			w.metrics.ChgoPoolMaxResources().Set(float64(stat.MaxResources()))
			w.metrics.ChgoPoolAcquireDuration().Set(stat.AcquireDuration().Seconds())
			w.metrics.ChgoPoolEmptyAcquireWaitTime().Set(stat.EmptyAcquireWaitTime().Seconds())

			acquireCount := stat.AcquireCount()
			if delta := acquireCount - prevAcquireCount; delta > 0 {
				w.metrics.ChgoPoolAcquireTotal().Add(float64(delta))
			}

			prevAcquireCount = acquireCount

			emptyAcquireCount := stat.EmptyAcquireCount()
			if delta := emptyAcquireCount - prevEmptyAcquireCount; delta > 0 {
				w.metrics.ChgoPoolEmptyAcquireTotal().Add(float64(delta))
			}

			prevEmptyAcquireCount = emptyAcquireCount

			canceledAcquireCount := stat.CanceledAcquireCount()
			if delta := canceledAcquireCount - prevCanceledAcquireCount; delta > 0 {
				w.metrics.ChgoPoolCanceledAcquireTotal().Add(float64(delta))
			}

			prevCanceledAcquireCount = canceledAcquireCount
		}
	}
}
