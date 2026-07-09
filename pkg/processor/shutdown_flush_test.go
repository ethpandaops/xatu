package processor

import (
	"context"
	"strconv"
	"testing"
	"time"
)

// TestShutdownFlushesFinalPartialBatch verifies that a graceful shutdown exports the
// final partial batch held in the batch builder rather than dropping it. The batch size
// and timeout are set so the items never flush on their own, so they sit in the builder
// until shutdown.
func TestShutdownFlushesFinalPartialBatch(t *testing.T) {
	const (
		trials      = 200
		itemsPerRun = 3 // below MaxExportBatchSize, so never flushed by size
	)

	for _, workers := range []int{1, 4} {
		workers := workers

		t.Run("workers="+strconv.Itoa(workers), func(t *testing.T) {
			lostItems := 0

			for i := 0; i < trials; i++ {
				be := testBatchExporter[TestItem]{}

				bsp, err := NewBatchItemProcessor[TestItem](
					&be, "processor", nullLogger(),
					WithMaxExportBatchSize(1000),
					WithBatchTimeout(1*time.Hour),
					WithWorkers(workers),
					WithShippingMethod(ShippingMethodAsync),
				)
				if err != nil {
					t.Fatalf("constructor: %v", err)
				}

				bsp.Start(context.Background())

				items := make([]*TestItem, itemsPerRun)
				for j := range items {
					items[j] = &TestItem{name: "x"}
				}

				if err := bsp.Write(context.Background(), items); err != nil {
					t.Fatalf("write: %v", err)
				}

				// Let the builder pull the items into its in-memory partial batch.
				time.Sleep(2 * time.Millisecond)

				if err := bsp.Shutdown(context.Background()); err != nil {
					t.Fatalf("shutdown: %v", err)
				}

				if got := be.len(); got < itemsPerRun {
					lostItems += itemsPerRun - got
				}
			}

			if lostItems > 0 {
				t.Fatalf("shutdown dropped %d items across %d trials; the final partial batch must be exported", lostItems, trials)
			}
		})
	}
}
